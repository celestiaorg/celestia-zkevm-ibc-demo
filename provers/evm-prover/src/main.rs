use std::env;
use std::fs;
use tonic::{transport::Server, Request, Response, Status};
// Import the generated proto rust code
pub mod prover {
    tonic::include_proto!("celestia.prover.v1");
}

use alloy::primitives::Address;
use prover::prover_server::{Prover, ProverServer};
use prover::{
    ProveMembershipRequest, ProveMembershipResponse, ProveStateTransitionRequest,
    ProveStateTransitionResponse,
};
use reqwest::Url;
use sp1_ics07_tendermint_prover::{
    programs::UpdateClientProgram,
    prover::{SP1ICS07TendermintProver, SupportedProofType},
};
use sp1_ics07_tendermint_utils::rpc::TendermintRpcExt;
use tendermint_rpc::HttpClient;

// TODO: swap tendermint prover for blevm sp1 prover
pub struct ProverService {
    tendermint_prover: SP1ICS07TendermintProver<UpdateClientProgram>,
    tendermint_rpc_client: HttpClient,
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
            evm_rpc_url: url,
            evm_contract_address: contract_address
                .parse()
                .expect("Failed to parse contract address"),
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

        // TODO: get blevm sp1 proof
        let response = ProveStateTransitionResponse {
            proof: vec![],
            public_values: vec![],
        };

        Ok(Response::new(response))
    }

    async fn prove_membership(
        &self,
        request: Request<ProveMembershipRequest>,
    ) -> Result<Response<ProveMembershipResponse>, Status> {
        println!("Got membership request: {:?}", request);

        // Implement your membership proof logic here
        let response = ProveMembershipResponse {
            proof: vec![],
            height: 0,
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
                .unwrap(),
        )
        .serve(addr)
        .await?;

    Ok(())
}
