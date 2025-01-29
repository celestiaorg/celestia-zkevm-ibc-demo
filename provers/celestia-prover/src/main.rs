use ibc_eureka_solidity_types::sp1_ics07::sp1_ics07_tendermint::clientStateReturn;
use ibc_eureka_solidity_types::sp1_ics07::sp1_ics07_tendermint::getClientStateReturn;
use sp1_sdk::HashableKey;
use std::env;
use std::error::Error;
use std::fs;
use tonic::{transport::Server, Request, Response, Status};
use std::path::PathBuf;
use celestia_prover::{
    programs::{MembershipProgram, UpdateClientProgram},
    prover::{SP1ICS07TendermintProver, SupportedProofType},
};
use alloy::primitives::Address;
use alloy::providers::ProviderBuilder;
use ibc_core_commitment_types::merkle::MerkleProof;
use ibc_eureka_solidity_types::sp1_ics07::{
    sp1_ics07_tendermint, IICS07TendermintMsgs::{ConsensusState, ClientState},
};
use reqwest::Url;
use sp1_ics07_tendermint_utils::{light_block::LightBlockExt, rpc::TendermintRpcExt};
use tendermint_rpc::HttpClient;

// Import the generated proto rust code
// pub mod prover {
//     tonic::include_proto!("celestia.prover.v1");
// }
// use prover::prover_server::{Prover, ProverServer};
// use prover::{
//     InfoRequest, InfoResponse, ProveStateMembershipRequest, ProveStateMembershipResponse,
//     ProveStateTransitionRequest, ProveStateTransitionResponse,
// };

#[tokio::main]
async fn main() {
    foo().await;
}

async fn foo() {
    println!("Hello, world!");
    dotenv::dotenv().ok();
    let rpc_url = env::var("RPC_URL").expect("RPC_URL not set");
    let url = Url::parse(rpc_url.as_str()).expect("Failed to parse RPC_URL");
    let client_id = "0x25cdbd2bf399341f8fee22ecdb06682ac81fdc37".parse::<Address>().map_err(|e| {
        Status::internal(format!("Failed to parse client_id as EVM address: {}", e))
    }).expect("failed to parse client_id as EVM address");
    println!("Client ID: {:?}", client_id);

    let provider = ProviderBuilder::new()
        .with_recommended_fillers()
        .on_http(url.clone());
    println!("Provider: {:?}", provider);

    let contract = sp1_ics07_tendermint::new(client_id, provider);
    println!("Contract: {:?}", contract);
    println!("contract address {}", contract.address());
    // contract.

    // Fetch the client state as Bytes
    let client_state_bytes = contract
        .getClientState()
        .call()
        .await
        .expect("Failed to fetch client state");
    println!("Got client state bytes {:?}", client_state_bytes);
}
