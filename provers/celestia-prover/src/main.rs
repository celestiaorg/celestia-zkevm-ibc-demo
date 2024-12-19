use tonic::{transport::Server, Request, Response, Status};
use std::fs;
use std::env;
// Import the generated proto rust code
pub mod prover {
    tonic::include_proto!("celestia.prover.v1");
}

use prover::prover_server::{Prover, ProverServer};
use prover::{
    ProveStateTransitionRequest, ProveStateTransitionResponse,
    ProveMembershipRequest, ProveMembershipResponse,
};
use sp1_ics07_tendermint_prover::{
    programs::{UpdateClientProgram, MembershipProgram},
    prover::{SP1ICS07TendermintProver, SupportedProofType},
};
use tendermint_rpc::HttpClient;
use sp1_ics07_tendermint_utils::{light_block::LightBlockExt, rpc::TendermintRpcExt};
use ibc_eureka_solidity_types::sp1_ics07::{
    IICS07TendermintMsgs::ConsensusState,
    sp1_ics07_tendermint,
};
use reqwest::Url;
use alloy::providers::ProviderBuilder;
use alloy::primitives::Address;
use ibc_core_commitment_types::merkle::MerkleProof;

 pub struct ProverService {
    tendermint_prover: SP1ICS07TendermintProver<UpdateClientProgram>,
    tendermint_rpc_client: HttpClient,
    membership_prover: SP1ICS07TendermintProver<MembershipProgram>,
    evm_rpc_url: Url,
    evm_contract_address: Address,
}

impl ProverService {
    fn new() -> ProverService {
        let rpc_url = env::var("RPC_URL").expect("RPC_URL not set");
        let contract_address = env::var("CONTRACT_ADDRESS").expect("CONTRACT_ADDRESS not set");
        let url = Url::parse(rpc_url.as_str()).expect("Failed to parse RPC_URL");

        ProverService {
            tendermint_prover: SP1ICS07TendermintProver::new(SupportedProofType::Groth16),
            tendermint_rpc_client: HttpClient::from_env(),
            membership_prover: SP1ICS07TendermintProver::new(SupportedProofType::Groth16),
            evm_rpc_url: url,
            evm_contract_address: contract_address.parse().expect("Failed to parse contract address"),
        }
    }
}

#[tonic::async_trait]
impl Prover for ProverService {
    async fn prove_state_transition(
        &self,
        request: Request<ProveStateTransitionRequest>,
    ) -> Result<Response<ProveStateTransitionResponse>, Status> {
        println!("Got state transition request: {:?}", request);

        let provider = ProviderBuilder::new()
            .with_recommended_fillers()
            .on_http(self.evm_rpc_url.clone());
        let contract = sp1_ics07_tendermint::new(self.evm_contract_address, provider);

        let client_state = contract.getClientState().call().await.map_err(|e| Status::internal(e.to_string()))?._0;
        // fetch the light block at the latest height of the client state
        let trusted_light_block = self.tendermint_rpc_client
            .get_light_block(Some(client_state.latestHeight.revisionHeight))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        // fetch the latest light block 
        let target_light_block = self.tendermint_rpc_client
            .get_light_block(None)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        let trusted_consensus_state: ConsensusState = trusted_light_block.to_consensus_state().into();
        let proposed_header = target_light_block.into_header(&trusted_light_block);

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_err(|e| Status::internal(e.to_string()))?
            .as_secs();

        println!("proving from height {:?} to height {:?}", &trusted_light_block.signed_header.header.height, &proposed_header.trusted_height);

        let proof = self.tendermint_prover.generate_proof(
            &client_state,
            &trusted_consensus_state,
            &proposed_header,
            now,
        );
        let response = ProveStateTransitionResponse {
            proof: proof.bytes().to_vec(),
            public_values: proof.public_values.to_vec(),
        };

        Ok(Response::new(response))
    }

    async fn prove_membership(
        &self,
        request: Request<ProveMembershipRequest>,
    ) -> Result<Response<ProveMembershipResponse>, Status> {
        println!("Got membership request: {:?}", request);
        let inner_request = request.into_inner();

        let trusted_block = self.tendermint_rpc_client.get_light_block(Some(inner_request.height as u32))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;


        let key_proofs: Vec<(Vec<Vec<u8>>, Vec<u8>, MerkleProof)> = 
            futures::future::try_join_all(inner_request.key_paths.into_iter().map(|path| async {
                let path = vec![b"ibc".into(), path.into_bytes()];

                let (value, proof) = self.tendermint_rpc_client.prove_path(&path, trusted_block.signed_header.header.height.value() as u32).await?;

                anyhow::Ok((path, value, proof))
            }))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        
        println!("Generating membership proof for app hash {:?}", &trusted_block.signed_header.header.app_hash.as_bytes());
        let proof = self.membership_prover.generate_proof(
            &trusted_block.signed_header.header.app_hash.as_bytes(),
            key_proofs,
        );

        // Implement your membership proof logic here
        let response = ProveMembershipResponse {
            proof: proof.bytes().to_vec(),
            height: trusted_block.signed_header.header.height.value() as i64,
        };

        Ok(Response::new(response))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenv::dotenv().ok();
    let addr = "[::1]:50051".parse()?;
    let prover = ProverService::new();

    println!("Prover Server listening on {}", addr);

    // Load the file descriptor set
    let file_descriptor_set = fs::read("proto_descriptor.bin")?;

    Server::builder()
        .add_service(ProverServer::new(prover))
        .add_service(
            tonic_reflection::server::Builder::configure()
                .register_encoded_file_descriptor_set(&file_descriptor_set)
                .build_v1()
                .unwrap()
        )
        .serve(addr)
        .await?;

    Ok(())
}