use sp1_sdk::HashableKey;
use std::env;
use std::fs;
use std::path::PathBuf;
use tonic::{transport::Server, Request, Response, Status};

// Import the generated proto rust code
pub mod prover {
    tonic::include_proto!("celestia.prover.v1");
    tonic::include_proto!("celestia.ibc.lightclients.groth16.v1");
}

use prover::prover_server::{Prover, ProverServer};
use prover::{
    InfoRequest, InfoResponse, ProveStateMembershipRequest, ProveStateMembershipResponse,
    ProveStateTransitionRequest, ProveStateTransitionResponse,
};
use sp1_sdk::{include_elf, ProverClient, SP1ProvingKey};

// The ELF file for the Succinct RISC-V zkVM.
const BLEVM_ELF: &[u8] = include_elf!("blevm-mock");
pub struct ProverService {
    sp1_proving_key: SP1ProvingKey,
}

impl ProverService {
    async fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let sp1_prover = ProverClient::from_env();
        let (pk, _) = sp1_prover.setup(&BLEVM_ELF);

        Ok(ProverService {
            sp1_proving_key: pk,
        })
    }
}

#[tonic::async_trait]
impl Prover for ProverService {
    async fn info(&self, _request: Request<InfoRequest>) -> Result<Response<InfoResponse>, Status> {
        let state_transition_verifier_key = self.sp1_proving_key.vk.bytes32();
        // Empty string membership verifier key because currently membership proofs are not supported
        let state_membership_verifier_key = String::new();
        let response = InfoResponse {
            state_membership_verifier_key,
            state_transition_verifier_key,
        };

        Ok(Response::new(response))
    }

    async fn prove_state_transition(
        &self,
        _request: Request<ProveStateTransitionRequest>,
    ) -> Result<Response<ProveStateTransitionResponse>, Status> {
        Err(Status::unimplemented(
            "State transition proofs not yet implemented",
        ))
    }

    async fn prove_state_membership(
        &self,
        _request: Request<ProveStateMembershipRequest>,
    ) -> Result<Response<ProveStateMembershipResponse>, Status> {
        Err(Status::unimplemented(
            "Membership proofs not yet implemented",
        ))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenv::dotenv().ok();

    let addr = "[::]:50052".parse()?;

    let prover = ProverService::new().await?;

    println!("BLEVM Prover Server listening on {}", addr);

    // Get the path to the proto descriptor file from the environment variable
    let proto_descriptor_path: String = env::var("EVM_PROTO_DESCRIPTOR_PATH")
        .expect("EVM_PROTO_DESCRIPTOR_PATH environment variable not set");

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
