use tonic::{transport::Server, Request, Response, Status};
use std::fs;
use std::env;
use ibc_client_tendermint_types::ConsensusState;
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
    sp1_ics07_tendermint,
};
use reqwest::Url;
use alloy::providers::ProviderBuilder;
use alloy::primitives::Address;

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
        let url = Url::parse(rpc_url.as_str()).expect("Failed to parse URL");

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

        let trusted_consensus_state = trusted_light_block.to_consensus_state().into();
        let proposed_header = target_light_block.into_header(&trusted_light_block);

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_err(|e| Status::internal(e.to_string()))?
            .as_secs();

        println!("proving from height {:?} to height {:?}", &trusted_consensus_state.height, &proposed_header.height);

        let proof = self.tendermint_prover.generate_proof(
            &client_state,
            &trusted_consensus_state,
            &proposed_header,
            now,
        );
        
        // Implement your state transition proof logic here
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

        // let light_block = self.tendermint_rpc_client.get_light_block(request.height).await.map_err(|e| Status::internal(e.to_string()))?;

        // let proof = self.membership_prover.generate_proof(
        //     &light_block.signed_header.state_root,
        //     request.kv_proofs,
        // );

        // Implement your membership proof logic here
        let response = ProveMembershipResponse {
            proof: vec![], // Replace with actual proof
            height: 0,
        };

        Ok(Response::new(response))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
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
                .build()
                .unwrap()
        )
        .serve(addr)
        .await?;

    Ok(())
}