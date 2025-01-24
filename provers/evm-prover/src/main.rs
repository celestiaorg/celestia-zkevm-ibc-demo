use ibc_proto::ibc::core::client::v1::QueryClientStateRequest;
use prost::Message;
use std::env;
use std::fs;
use tonic::{transport::Server, Request, Response, Status};

// Import the generated proto rust code
pub mod prover {
    tonic::include_proto!("celestia.prover.v1");
    tonic::include_proto!("celestia.ibc.lightclients.groth16.v1");
}

use prover::prover_server::{Prover, ProverServer};
use prover::{
    ClientState, InfoRequest, InfoResponse, ProveStateMembershipRequest,
    ProveStateMembershipResponse, ProveStateTransitionRequest, ProveStateTransitionResponse,
};

use celestia_rpc::{BlobClient, Client, HeaderClient};
use celestia_types::{nmt::Namespace, Blob};
use rsp_client_executor::io::ClientExecutorInput;
use sp1_sdk::{include_elf, ProverClient, SP1Stdin};

use ethers::{
    providers::{Http, Middleware, Provider},
    types::BlockNumber,
};

use ibc_proto::ibc::core::client::v1::query_client::QueryClient as ClientQueryClient;

/// The ELF file for the Succinct RISC-V zkVM.
const BLEVM_ELF: &[u8] = include_elf!("blevm");

pub struct ProverService {
    celestia_client: Client,
    evm_client: Provider<Http>,
    sp1_prover: ProverClient,
    simapp_client: ClientQueryClient<tonic::transport::Channel>,
    namespace: Namespace,
}

impl ProverService {
    async fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let token = env::var("CELESTIA_NODE_AUTH_TOKEN").expect("Token not provided");
        let namespace_str = env::var("CELESTIA_NAMESPACE").expect("Namespace not provided");

        let namespace = Namespace::new_v0(&alloy::hex::decode(namespace_str)?)
            .map_err(|e| Status::internal(format!("Failed to create namespace: {}", e)))?;

        let celestia_client = Client::new("ws://localhost:26658", Some(&token))
            .await
            .expect("Failed creating Celestia RPC client");

        let evm_rpc = env::var("EVM_RPC_URL").expect("EVM_RPC_URL not provided");
        let evm_client = Provider::try_from(evm_rpc)?;
        let simapp_rpc = env::var("SIMAPP_RPC_URL").expect("SIMAPP_RPC_URL not provided");
        let simapp_client = ClientQueryClient::connect(simapp_rpc).await?;

        Ok(ProverService {
            celestia_client,
            evm_client,
            sp1_prover: ProverClient::new(),
            simapp_client,
            namespace,
        })
    }

    async fn get_latest_height(&self) -> Result<u64, Status> {
        self.evm_client
            .get_block(BlockNumber::Latest)
            .await
            .map_err(|e| Status::internal(format!("Failed to get latest block: {}", e)))?
            .ok_or_else(|| Status::internal("No latest block found"))?
            .number
            .ok_or_else(|| Status::internal("Block has no number"))?
            .as_u64()
            .try_into()
            .map_err(|e| Status::internal(format!("Failed to convert block number: {}", e)))
    }

    async fn get_trusted_height(&self, client_id: &str) -> Result<u64, Status> {
        let request = tonic::Request::new(QueryClientStateRequest {
            client_id: client_id.to_string(),
        });

        let response = self
            .simapp_client
            .clone()
            .client_state(request)
            .await?
            .into_inner();

        let client_state_json = response
            .client_state
            .ok_or_else(|| Status::internal("Failed to query client state"))?;
        let client_state = ClientState::decode(client_state_json.value.as_slice())
            .map_err(|e| Status::internal(format!("Failed to decode client state: {}", e)))?;
        Ok(client_state.latest_height)
    }
}

#[tonic::async_trait]
impl Prover for ProverService {
    async fn info(&self, _request: Request<InfoRequest>) -> Result<Response<InfoResponse>, Status> {
        let response = InfoResponse {
            state_membership_verifier_key: vec![],
            state_transition_verifier_key: vec![],
        };

        Ok(Response::new(response))
    }

    async fn prove_state_transition(
        &self,
        request: Request<ProveStateTransitionRequest>,
    ) -> Result<Response<ProveStateTransitionResponse>, Status> {
        println!("Got state transition request: {:?}", request);

        let inner_request = request.into_inner();

        // Get the latest height from EVM rollup.
        let latest_height = self.get_latest_height().await?;

        // Get the trusted height from groth16 client.
        let trusted_height = self
            .get_trusted_height(inner_request.client_id.as_str())
            .await?;

        println!(
            "proving from height {:?} to height {:?}",
            latest_height, trusted_height
        );

        // TODO: Generate aggregate proofs for blocks in the range [trusted_height, latest_height]
        // blocked on blevm.generate_aggregate_proofs(trusted_height, latest_height)

        // Currently this uses a fixture to create a single block proof
        // Load the zkEVM input from the file
        let input_bytes = fs::read(format!("input/1/{}.bin", latest_height))
            .map_err(|e| Status::internal(format!("Failed to read input file: {}", e)))?;

        let input: ClientExecutorInput = bincode::deserialize(&input_bytes)
            .map_err(|e| Status::internal(format!("Failed to deserialize input: {}", e)))?;

        // Create blob from the EVM block
        let block = input.current_block.clone();
        let block_bytes = bincode::serialize(&block)
            .map_err(|e| Status::internal(format!("Failed to serialize block: {}", e)))?;

        let blob = Blob::new(self.namespace, block_bytes, celestia_types::AppVersion::V3)
            .map_err(|e| Status::internal(format!("Failed to create blob: {}", e)))?;

        // Fetch the blob from the chain to get its index
        let blob_from_chain = self
            .celestia_client
            .blob_get(latest_height, self.namespace, blob.commitment.clone())
            .await
            .map_err(|e| Status::internal(format!("Failed to get blob: {}", e)))?;

        // Get the header and retrieve the EDS roots
        let header = self
            .celestia_client
            .header_get_by_height(latest_height)
            .await
            .map_err(|e| Status::internal(format!("Failed to get header: {}", e)))?;

        let eds_row_roots = header.dah.row_roots();
        let _eds_column_roots = header.dah.column_roots();
        let eds_size: u64 = eds_row_roots.len().try_into().unwrap();
        let ods_size = eds_size / 2;

        // Calculate blob indices
        let blob_index: u64 = blob_from_chain.index.unwrap();
        let blob_size: u64 = std::cmp::max(1, blob_from_chain.to_shares().unwrap().len() as u64);
        let first_row_index: u64 = blob_index.div_ceil(eds_size) - 1;
        let ods_index = blob_from_chain.index.unwrap() - (first_row_index * ods_size);
        let last_row_index: u64 = (ods_index + blob_size).div_ceil(ods_size) - 1;

        // Get NMT proofs
        let nmt_multiproofs = self
            .celestia_client
            .blob_get_proof(latest_height, self.namespace, blob.commitment.clone())
            .await
            .map_err(|e| Status::internal(format!("Failed to get proof: {}", e)))?;

        // Setup SP1 inputs
        let mut stdin = SP1Stdin::new();
        stdin.write(&input);
        stdin.write(&self.namespace);
        stdin.write(&header.header.hash());
        stdin.write(&header.dah);
        stdin.write(&nmt_multiproofs);
        stdin.write(
            &eds_row_roots[first_row_index as usize..(last_row_index + 1) as usize].to_vec(),
        );

        // Generate proof
        let (pk, _) = self.sp1_prover.setup(&BLEVM_ELF);
        let proof = self
            .sp1_prover
            .prove(&pk, stdin)
            .core()
            .run()
            .map_err(|e| Status::internal(format!("Failed to generate proof: {}", e)))?;

        let response = ProveStateTransitionResponse {
            proof: bincode::serialize(&proof).unwrap(),
            public_values: vec![],
        };

        Ok(Response::new(response))
    }

    async fn prove_state_membership(
        &self,
        _request: Request<ProveStateMembershipRequest>,
    ) -> Result<Response<ProveStateMembershipResponse>, Status> {
        // TODO: Implement membership proofs later
        Err(Status::unimplemented(
            "Membership proofs not yet implemented",
        ))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenv::dotenv().ok();

    let addr = "[::1]:50051".parse()?;
    let prover = ProverService::new().await?;

    println!("BLEVM Prover Server listening on {}", addr);

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
