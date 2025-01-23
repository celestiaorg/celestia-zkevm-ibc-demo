use sp1_sdk::HashableKey;
use std::env;
use std::fs;
use tonic::{transport::Server, Request, Response, Status};
// Import the generated proto rust code
pub mod prover {
    tonic::include_proto!("celestia.prover.v1");
}
use std::path::PathBuf;

use prover::prover_server::{Prover, ProverServer};
use prover::{
    InfoRequest, InfoResponse, ProveStateMembershipRequest, ProveStateMembershipResponse,
    ProveStateTransitionRequest, ProveStateTransitionResponse,
};

use celestia_prover::{
    programs::{MembershipProgram, UpdateClientProgram},
    prover::{SP1ICS07TendermintProver, SupportedProofType},
};

use alloy::primitives::Address;
use alloy::providers::ProviderBuilder;
use ibc_core_commitment_types::merkle::MerkleProof;
use ibc_eureka_solidity_types::sp1_ics07::{
    sp1_ics07_tendermint, IICS07TendermintMsgs::ConsensusState,
};
use reqwest::Url;
use sp1_ics07_tendermint_utils::{light_block::LightBlockExt, rpc::TendermintRpcExt};
use tendermint_rpc::HttpClient;

pub struct ProverService {
    tendermint_prover: SP1ICS07TendermintProver<UpdateClientProgram>,
    tendermint_rpc_client: HttpClient,
    membership_prover: SP1ICS07TendermintProver<MembershipProgram>,
    evm_rpc_url: Url,
}

impl ProverService {
    fn new() -> ProverService {
        let rpc_url = env::var("RPC_URL").expect("RPC_URL not set");
        let url = Url::parse(rpc_url.as_str()).expect("Failed to parse RPC_URL");

        ProverService {
            tendermint_prover: SP1ICS07TendermintProver::new(SupportedProofType::Groth16),
            tendermint_rpc_client: HttpClient::from_env(),
            membership_prover: SP1ICS07TendermintProver::new(SupportedProofType::Groth16),
            evm_rpc_url: url,
        }
    }
}

#[tonic::async_trait]
impl Prover for ProverService {
    async fn info(&self, _request: Request<InfoRequest>) -> Result<Response<InfoResponse>, Status> {
        let state_transition_verifier_key =
            bincode::serialize(&self.tendermint_prover.vkey.hash_bytes())
                .map_err(|e| Status::internal(e.to_string()))?;
        let state_membership_verifier_key =
            bincode::serialize(&self.membership_prover.vkey.hash_bytes())
                .map_err(|e| Status::internal(e.to_string()))?;
        let response = InfoResponse {
            state_transition_verifier_key,
            state_membership_verifier_key,
        };

        Ok(Response::new(response))
    }

    async fn prove_state_transition(
        &self,
        request: Request<ProveStateTransitionRequest>,
    ) -> Result<Response<ProveStateTransitionResponse>, Status> {
        println!("Got state transition request: {:?}", request);
        let inner_request = request.into_inner();

        let client_id = inner_request.client_id.parse::<Address>().map_err(|e| {
            Status::internal(format!("Failed to parse client_id as EVM address: {}", e))
        })?;

        let provider = ProviderBuilder::new()
            .with_recommended_fillers()
            .on_http(self.evm_rpc_url.clone());
        let contract = sp1_ics07_tendermint::new(client_id, provider);

        let client_state = contract
            .getClientState()
            .call()
            .await
            .map_err(|e| Status::internal(e.to_string()))?
            ._0;
        // fetch the light block at the latest height of the client state
        let trusted_light_block = self
            .tendermint_rpc_client
            .get_light_block(Some(client_state.latestHeight.revisionHeight))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        // fetch the latest light block
        let target_light_block = self
            .tendermint_rpc_client
            .get_light_block(None)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        let trusted_consensus_state: ConsensusState =
            trusted_light_block.to_consensus_state().into();
        let proposed_header = target_light_block.into_header(&trusted_light_block);

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_err(|e| Status::internal(e.to_string()))?
            .as_secs();

        println!(
            "proving from height {:?} to height {:?}",
            &trusted_light_block.signed_header.header.height, &proposed_header.trusted_height
        );

        let proof = self.tendermint_prover.generate_proof(
            &client_state,
            &trusted_consensus_state,
            &proposed_header,
            now,
        );
        println!("generated proof {:?}", proof.bytes());
        println!(
            "generated proof public values {:?}",
            proof.public_values.as_slice().to_vec()
        );

        let response = ProveStateTransitionResponse {
            proof: proof.bytes().to_vec(),
            public_values: proof.public_values.as_slice().to_vec(),
        };

        Ok(Response::new(response))
    }

    async fn prove_state_membership(
        &self,
        request: Request<ProveStateMembershipRequest>,
    ) -> Result<Response<ProveStateMembershipResponse>, Status> {
        println!("Got state membership request...");
        let inner_request = request.into_inner();

        let trusted_block = self
            .tendermint_rpc_client
            .get_light_block(Some(inner_request.height as u32))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        let key_proofs: Vec<(Vec<Vec<u8>>, Vec<u8>, MerkleProof)> =
            futures::future::try_join_all(inner_request.key_paths.into_iter().map(|path| async {
                let path = vec![b"ibc".into(), path.into_bytes()];

                let (value, proof) = self
                    .tendermint_rpc_client
                    .prove_path(
                        &path,
                        trusted_block.signed_header.header.height.value() as u32,
                    )
                    .await?;

                anyhow::Ok((path, value, proof))
            }))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        let proof = self.membership_prover.generate_proof(
            trusted_block.signed_header.header.app_hash.as_bytes(),
            key_proofs,
        );

        println!(
            "Generated membership proof for height: {:?}",
            trusted_block.signed_header.header.height.value() as i64
        );

        // Implement your membership proof logic here
        let response = ProveStateMembershipResponse {
            proof: proof.bytes().to_vec(),
            height: trusted_block.signed_header.header.height.value() as i64,
        };

        Ok(Response::new(response))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenv::dotenv().ok();
    let addr = "[::]:50051".parse()?;
    let prover = ProverService::new();

    println!("Prover Server listening on {}", addr);

    // Get the path to the proto descriptor file from the environment variable
    let proto_descriptor_path = env::var("PROTO_DESCRIPTOR_PATH")
        .expect("PROTO_DESCRIPTOR_PATH environment variable not set");

    println!(
        "Loading proto descriptor set from {}",
        proto_descriptor_path
    );
    let file_path = PathBuf::from(proto_descriptor_path);

    // Read the file
    let file_descriptor_set = fs::read(&file_path)?;
    println!("Loaded proto descriptor set");

    Server::builder()
        .add_service(ProverServer::new(prover))
        .add_service(
            tonic_reflection::server::Builder::configure()
                .register_encoded_file_descriptor_set(&file_descriptor_set)
                .build_v1()
                .unwrap(),
        )
        .serve(addr)
        .await?;

    Ok(())
}
