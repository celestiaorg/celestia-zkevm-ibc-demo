use blevm_prover::indexer::get_inclusion_height;
use blevm_prover::prover::{
    AggregatorConfig, BlockProver, BlockProverInput, CelestiaClient, CelestiaConfig, ProverConfig,
};
use blevm_prover::rsp::generate_client_input;
use ibc_proto::ibc::core::client::v1::QueryClientStateRequest;
use prost::Message;
use rsp_primitives::genesis::Genesis;
use sp1_sdk::{include_elf, HashableKey, ProverClient, SP1VerifyingKey};
use std::env;
use std::fs;
use std::path::PathBuf;
use tendermint::merkle;
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

use celestia_types::nmt::Namespace;
use celestia_types::Commitment;

use ethers::{
    providers::{Http, Middleware, Provider},
    types::BlockNumber,
};

use ibc_proto::ibc::core::client::v1::query_client::QueryClient as ClientQueryClient;

pub const BLEVM_ELF: &[u8] = include_elf!("blevm");
pub const BLEVM_AGGREGATOR_ELF: &[u8] = include_elf!("blevm-aggregator");

pub struct ProverService {
    evm_rpc_url: String,
    evm_client: Provider<Http>,
    prover: BlockProver,
    simapp_client: ClientQueryClient<tonic::transport::Channel>,
    indexer_url: String,
    genesis: Genesis,
    custom_beneficiary: Option<String>,
    opcode_tracking: bool,
    aggregator_vkey: SP1VerifyingKey,
}

impl ProverService {
    async fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let evm_rpc_url = env::var("EVM_RPC_URL").expect("EVM_RPC_URL not provided");
        let evm_client = Provider::try_from(evm_rpc_url.clone())?;
        let simapp_rpc = env::var("SIMAPP_RPC_URL").expect("SIMAPP_RPC_URL not provided");
        let simapp_client = ClientQueryClient::connect(simapp_rpc).await?;
        let indexer_url = env::var("INDEXER_URL").expect("INDEXER_URL not provided");

        let prover_config = ProverConfig {
            elf_bytes: BLEVM_ELF,
        };

        let aggregator_config = AggregatorConfig {
            elf_bytes: BLEVM_AGGREGATOR_ELF,
        };

        let celestia_config = CelestiaConfig {
            node_url: std::env::var("CELESTIA_NODE_URL").expect("CELESTIA_NODE_URL must be set"),
            auth_token: std::env::var("CELESTIA_NODE_AUTH_TOKEN")
                .expect("CELESTIA_NODE_AUTH_TOKEN must be set"),
        };
        let namespace_hex =
            std::env::var("CELESTIA_NAMESPACE").expect("CELESTIA_NAMESPACE must be set");
        let namespace = Namespace::new_v0(&hex::decode(namespace_hex)?)?;
        let celestia_client = CelestiaClient::new(celestia_config, namespace).await?;

        // Create the SP1 client once
        let sp1_client = ProverClient::from_env();

        // Setup keys using the single client instance
        let (_, aggregator_vkey) = sp1_client.setup(BLEVM_AGGREGATOR_ELF);

        // Pass the sp1_client to BlockProver::new
        let prover = BlockProver::new(
            celestia_client,
            prover_config,
            aggregator_config,
            sp1_client,
        );

        let custom_beneficiary = env::var("CUSTOM_BENEFICIARY").ok();
        let opcode_tracking = env::var("OPCODE_TRACKING").is_ok();

        let genesis_path = std::env::var("GENESIS_PATH").expect("GENESIS_PATH must be set");
        let genesis_json = fs::read_to_string(&genesis_path)?;

        let genesis = Genesis::Custom(genesis_json);

        Ok(ProverService {
            evm_rpc_url,
            evm_client,
            prover,
            simapp_client,
            indexer_url,
            genesis,
            custom_beneficiary,
            opcode_tracking,
            aggregator_vkey,
        })
    }

    async fn get_latest_height(&self) -> Result<ethers::types::U64, Status> {
        self.evm_client
            .get_block(BlockNumber::Latest)
            .await
            .map_err(|e| Status::internal(format!("Failed to get latest block: {}", e)))?
            .ok_or_else(|| Status::internal("No latest block found"))?
            .number
            .ok_or_else(|| Status::internal("Block has no number"))
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
            state_membership_verifier_key: "".to_string(),
            state_transition_verifier_key: hex::encode(self.aggregator_vkey.bytes32()),
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

        if latest_height.as_u64() <= trusted_height {
            return Err(Status::unimplemented(
                "Trusted height is greater than latest height",
            ));
        }

        let mut inputs = vec![];
        let start_height = trusted_height + 1;
        let end_height = latest_height.as_u64();

        println!(
            "proving from height {:?} to height {:?}",
            start_height, end_height
        );

        for height in start_height..=end_height {
            let (inclusion_height, blob_commitment) =
                get_inclusion_height(self.indexer_url.clone(), height)
                    .await
                    .unwrap();
            let client_executor_input = generate_client_input(
                self.evm_rpc_url.clone(),
                height,
                &self.genesis,
                self.custom_beneficiary.as_ref(),
                self.opcode_tracking,
            )
            .await
            .unwrap();
            let hash: merkle::Hash = blob_commitment[..blob_commitment.len()].try_into().unwrap();
            let commitment = Commitment::new(hash);
            let rollup_block = self
                .prover
                .get_blob(inclusion_height, commitment)
                .await
                .unwrap();

            let input = BlockProverInput {
                inclusion_height,
                client_executor_input,
                rollup_block,
            };
            inputs.push(input);
        }

        println!("generating aggregation proof, size: {}", inputs.len());

        let aggregation_output: blevm_prover::prover::AggregationOutput =
            self.prover.prove_block_range(inputs).await.unwrap();

        let response = ProveStateTransitionResponse {
            proof: bincode::serialize(&aggregation_output.proof.proof).unwrap(),
            public_values: aggregation_output.proof.public_values.to_vec(),
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
    if let Ok(mode) = env::var("SP1_PROVER") {
        println!("SP1_Prover mode: {mode}");
    } else {
        println!("SP1_Prover mode: undefined");
    };
    let addr = "[::]:50052".parse()?;
    let prover = ProverService::new().await?;

    println!("Prover Server listening on {}", addr);

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
