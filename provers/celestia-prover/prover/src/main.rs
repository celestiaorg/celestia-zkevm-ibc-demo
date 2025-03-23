use alloy::primitives::Bytes;
use alloy_sol_types::SolValue;
use ibc_eureka_solidity_types::sp1_ics07::{
    IICS07TendermintMsgs::ClientState,
    IMembershipMsgs::{KVPair, MembershipOutput, MembershipProof, SP1MembershipProof},
    ISP1Msgs::SP1Proof,
};
use sp1_sdk::HashableKey;
use std::env;
use std::fs;
use tonic::{transport::Server, Request, Response, Status};
pub mod prover {
    tonic::include_proto!("celestia.prover.v1");
}
use alloy::primitives::Address;
use alloy::providers::ProviderBuilder;
use celestia_prover::{
    programs::{MembershipProgram, UpdateClientProgram},
    prover::{SP1ICS07TendermintProver, SupportedProofType},
};
use ibc_core_commitment_types::merkle::MerkleProof;
use ibc_eureka_solidity_types::sp1_ics07::sp1_ics07_tendermint;
use ibc_eureka_solidity_types::sp1_ics07::IICS07TendermintMsgs::ConsensusState as SolConsensusState;
use prover::prover_server::{Prover, ProverServer};
use prover::{
    InfoRequest, InfoResponse, ProveStateMembershipRequest, ProveStateMembershipResponse,
    ProveStateTransitionRequest, ProveStateTransitionResponse,
};
use reqwest::Url;
use sp1_ics07_tendermint_utils::{light_block::LightBlockExt, rpc::TendermintRpcExt};
use std::path::PathBuf;
use std::time::Instant;
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
        let state_transition_verifier_key = self.tendermint_prover.vkey.bytes32();
        let state_membership_verifier_key = self.membership_prover.vkey.bytes32();
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
        println!("Got state transition request.");
        let inner_request = request.into_inner();

        let client_id = inner_request.client_id.parse::<Address>().map_err(|e| {
            Status::internal(format!("Failed to parse client_id as EVM address: {}", e))
        })?;
        println!("client_id: {:?}", client_id);

        let provider = ProviderBuilder::new()
            .with_recommended_fillers()
            .on_http(self.evm_rpc_url.clone());

        let contract = sp1_ics07_tendermint::new(client_id, provider);
        println!("contract: {:?}", contract);

        // Fetch the client state as Bytes
        let client_state_bytes = contract
            .getClientState()
            .call()
            .await
            .map_err(|e| Status::internal(e.to_string()))?
            ._0;
        println!("client_state_bytes: {:?}", client_state_bytes);

        let client_state =
            <ClientState as alloy_sol_types::SolType>::abi_decode(&client_state_bytes, true)
                .map_err(|e| Status::internal(e.to_string()))?;
        println!("client_state chainId: {:?}", client_state.chainId);

        // Fetch the block at the latest height of the client state. This is the
        // beginning of the range we need to prove.
        let trusted_block = self
            .tendermint_rpc_client
            .get_light_block(Some(client_state.latestHeight.revisionHeight))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        println!("trusted_block.height: {:?}", trusted_block.height());

        // Fetch the latest block on the chain. This is the end of the range we
        // need to prove.
        let target_block = self
            .tendermint_rpc_client
            .get_light_block(None)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        println!("target_block.height: {:?}", target_block.height());

        let trusted_consensus_state: SolConsensusState = trusted_block.to_consensus_state().into();
        println!("trusted_consensus_state: {:?}", trusted_consensus_state);

        let header = target_block.into_header(&trusted_block);

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_err(|e| Status::internal(e.to_string()))?
            .as_secs();

        println!(
            "proving from height {:?} to height {:?}",
            &header.trusted_height.revision_height(),
            &header.height().revision_height()
        );

        let start_time = Instant::now();
        let proof = self.tendermint_prover.generate_proof(
            &client_state,
            &trusted_consensus_state,
            &header,
            now,
        );
        let elapsed = start_time.elapsed();

        let response = ProveStateTransitionResponse {
            proof: proof.bytes().to_vec(),
            public_values: proof.public_values.to_vec(),
        };

        println!("Generated state transition proof in {:.2?}", elapsed);

        Ok(Response::new(response))
    }

    async fn prove_state_membership(
        &self,
        request: Request<ProveStateMembershipRequest>,
    ) -> Result<Response<ProveStateMembershipResponse>, Status> {
        let inner_request = request.into_inner();
        println!(
            "Got state membership request for height {:?} key paths {:?}...",
            inner_request.height, inner_request.key_paths
        );
        let trusted_block = self
            .tendermint_rpc_client
            .get_light_block(Some(inner_request.height as u32))
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        let trusted_consensus_state: SolConsensusState = trusted_block.to_consensus_state().into();
        println!("trusted_consensus_state: {:?}", trusted_consensus_state);

        let path_value_and_proofs: Vec<(Vec<Vec<u8>>, Vec<u8>, MerkleProof)> =
            futures::future::try_join_all(inner_request.key_paths.into_iter().map(|path| async {
                let path = vec![b"ibc".into(), path.into_bytes()];
                println!("path: {:?}", path);

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

        println!(
            "Generating proof with path_value_and_proofs: {:?}",
            path_value_and_proofs
        );

        let kv_proofs = path_value_and_proofs
            .into_iter()
            .map(|(path, value, proof)| {
                let path = path.into_iter().map(Bytes::from).collect::<Vec<Bytes>>();
                let value = Bytes::from(value);
                (KVPair { path, value }, proof)
            })
            .collect();

        // Generate the SP1 proof
        let start_time = Instant::now();
        let sp1_proof = self.membership_prover.generate_proof(
            trusted_block.signed_header.header.app_hash.as_bytes(),
            kv_proofs,
        );
        let elapsed = start_time.elapsed();

        println!(
            "Generated proof for height: {:?} in {:.2?}",
            trusted_block.signed_header.header.height.value() as i64,
            elapsed
        );

        let membership_output =
            MembershipOutput::abi_decode(&sp1_proof.public_values.to_vec(), true).unwrap();
        membership_output.kvPairs.iter().for_each(|kv| {
            println!(
                "membership_output path: {:?} value: {:?}",
                kv.path, kv.value
            );
        });

        let membership_proof = MembershipProof::from(SP1MembershipProof {
            sp1Proof: SP1Proof::new(
                &self.membership_prover.vkey.bytes32(),
                sp1_proof.bytes(),
                sp1_proof.public_values.to_vec(),
            ),
            trustedConsensusState: trusted_consensus_state,
        });

        println!(
            "Converted SP1 proof to membership_proof: {:?}",
            membership_proof
        );
        let proof = membership_proof.abi_encode().to_vec();

        let response = ProveStateMembershipResponse {
            proof,
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
    let proto_descriptor_path = env::var("CELESTIA_PROTO_DESCRIPTOR_PATH")
        .expect("CELESTIA_PROTO_DESCRIPTOR_PATH environment variable not set");

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
