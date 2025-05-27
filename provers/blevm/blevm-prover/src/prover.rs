use crate::proofs::generate_header_proofs;
use celestia_rpc::{BlobClient, Client, HeaderClient, ShareClient};
use celestia_types::{
    nmt::{Namespace, NamespaceProof},
    ExtendedHeader,
};
use celestia_types::{AppVersion, Blob, Commitment};
use eq_common::KeccakInclusionToDataRootProofInput;
use rsp_client_executor::io::EthClientExecutorInput;
use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use sp1_sdk::Prover;
use sp1_sdk::{
    ExecutionReport, HashableKey, SP1Proof, SP1ProofWithPublicValues, SP1PublicValues, SP1Stdin,
    SP1VerifyingKey,
};
use std::error::Error;

/// Configuration for the Celestia client
#[derive(Clone)]
pub struct CelestiaConfig {
    pub node_url: String,
    pub auth_token: String,
}

/// Configuration for the proof generation
pub struct ProverConfig {
    pub elf_bytes: &'static [u8],
}

/// Configuration for the aggregator
pub struct AggregatorConfig {
    pub elf_bytes: &'static [u8],
}

/// Input for proof aggregation
#[derive(Clone, Serialize, Deserialize)]
pub struct AggregationInput {
    pub proof: SP1ProofWithPublicValues,
    pub vk: SP1VerifyingKey,
}

/// Output from proof aggregation
pub struct AggregationOutput {
    /// The aggregated proof
    pub proof: SP1ProofWithPublicValues,
}

/// Input data for block proving
#[derive(Clone)]
pub struct BlockProverInput {
    pub inclusion_height: u64,
    pub client_executor_input: Vec<u8>,
    pub rollup_block: Vec<u8>,
}

// CelestiaClient wraps the client and implements helpers for querying celestia
pub struct CelestiaClient {
    client: Client,
    namespace: Namespace,
}

impl CelestiaClient {
    /// Creates a new CelestiaClient with the provided configuration and namespace
    ///
    /// # Arguments
    ///
    /// * `config` - Configuration for connecting to a Celestia node
    /// * `namespace` - The namespace to use for operations
    ///
    /// # Returns
    ///
    /// A new CelestiaClient instance or an error if connection fails
    pub async fn new(config: CelestiaConfig, namespace: Namespace) -> Result<Self, Box<dyn Error>> {
        let client = Client::new(&config.node_url, Some(&config.auth_token))
            .await
            .map_err(|e| format!("Failed creating RPC client: {}", e))?;

        Ok(Self { client, namespace })
    }

    /// Retrieves a blob from Celestia network at the specified height with given commitment
    ///
    /// # Arguments
    ///
    /// * `height` - The block height to query
    /// * `blob_commitment` - The commitment hash of the blob to retrieve
    ///
    /// # Returns
    ///
    /// The requested blob or an error if retrieval fails
    pub async fn get_blob(
        &self,
        height: u64,
        blob_commitment: &Commitment,
    ) -> Result<Blob, Box<dyn Error>> {
        let blob = self
            .client
            .blob_get(height, self.namespace, blob_commitment.clone())
            .await
            .map_err(|e| format!("Failed getting blob: {}", e))?;

        Ok(blob)
    }

    /// Retrieves an extended header from Celestia network at the specified height
    ///
    /// # Arguments
    ///
    /// * `height` - The block height to query
    ///
    /// # Returns
    ///
    /// The extended header at the specified height or an error if retrieval fails
    pub async fn get_header(&self, height: u64) -> Result<ExtendedHeader, Box<dyn Error>> {
        let header = self
            .client
            .header_get_by_height(height)
            .await
            .map_err(|e| format!("Failed getting header: {}", e))?;

        Ok(header)
    }

    /// Retrieves Namespace Merkle Tree (NMT) proofs for a blob at the specified height
    ///
    /// # Arguments
    ///
    /// * `height` - The block height to query
    /// * `blob` - The blob for which to retrieve proofs
    ///
    /// # Returns
    ///
    /// Vector of namespace proofs or an error if retrieval fails
    pub async fn get_nmt_proofs(
        &self,
        height: u64,
        blob: &Blob,
    ) -> Result<Vec<NamespaceProof>, Box<dyn Error>> {
        Ok(self
            .client
            .blob_get_proof(height, self.namespace, blob.commitment)
            .await
            .map_err(|e| format!("Failed getting NMT proofs: {}", e))?)
    }
}

/// Main prover service that coordinates the entire proving process
pub struct BlockProver {
    celestia_client: CelestiaClient,
    prover_config: ProverConfig,
    aggregator_config: AggregatorConfig,
    sp1_client: sp1_sdk::EnvProver,
}

impl BlockProver {
    /// Creates a new BlockProver with the provided client and configuration
    ///
    /// # Arguments
    ///
    /// * `celestia_client` - Client for interacting with Celestia network
    /// * `prover_config` - Configuration for the prover
    /// * `aggregator_config` - Configuration for proof aggregation
    /// * `sp1_client` - The SP1 prover client instance
    ///
    /// # Returns
    ///
    /// A new BlockProver instance
    pub fn new(
        celestia_client: CelestiaClient,
        prover_config: ProverConfig,
        aggregator_config: AggregatorConfig,
        sp1_client: sp1_sdk::EnvProver,
    ) -> Self {
        Self {
            celestia_client,
            prover_config,
            aggregator_config,
            sp1_client,
        }
    }

    /// Prepares SP1 standard input for proof generation
    ///
    /// # Arguments
    ///
    /// * `input` - Block prover input data containing inclusion height and block data
    ///
    /// # Returns
    ///
    /// Prepared SP1Stdin or an error
    async fn get_stdin(&self, input: BlockProverInput) -> Result<SP1Stdin, Box<dyn Error>> {
        let client_executor_input: EthClientExecutorInput =
            bincode::deserialize(&input.client_executor_input)?;
        let blob = Blob::new(
            self.celestia_client.namespace,
            input.rollup_block.clone(),
            AppVersion::V3,
        )?;

        let header = self
            .celestia_client
            .get_header(input.inclusion_height)
            .await?;

        let eds_row_roots = header.dah.row_roots();
        let eds_size: u64 = eds_row_roots.len().try_into().unwrap();
        let ods_size: u64 = eds_size / 2;

        // Get blob and header from Celestia
        let blob_from_chain = self
            .celestia_client
            .get_blob(input.inclusion_height, &blob.commitment)
            .await?;

        let _index = blob_from_chain.index.unwrap();
        //let first_row_index: u64 = index.div_ceil(eds_size) - 1;
        // Trying this Diego's way
        let first_row_index: u64 = blob_from_chain.index.unwrap() / eds_size;
        let ods_index = blob_from_chain.index.unwrap() - (first_row_index * ods_size);

        let range_response = self
            .celestia_client
            .client
            .share_get_range(&header, ods_index, ods_index + blob.shares_len() as u64)
            .await
            .expect("Failed getting shares");

        range_response
            .proof
            .verify(header.dah.hash())
            .expect("Failed verifying proof");

        let keccak_hash: [u8; 32] = Keccak256::new().chain_update(&blob.data).finalize().into();

        let proof_input = KeccakInclusionToDataRootProofInput {
            data: blob.clone().data,
            namespace_id: self.celestia_client.namespace,
            share_proofs: range_response.clone().proof.share_proofs,
            row_proof: range_response.clone().proof.row_proof,
            data_root: header.dah.hash().as_bytes().try_into().unwrap(),
            keccak_hash,
        };

        // Generate all required proofs
        let (data_hash_bytes, data_hash_proof) = generate_header_proofs(&header)?;

        // Prepare stdin for the prover
        let mut stdin = SP1Stdin::new();
        stdin.write(&proof_input);
        stdin.write(&client_executor_input);
        stdin.write(&header.header.hash());
        stdin.write_vec(data_hash_bytes);
        stdin.write(&data_hash_proof);
        Ok(stdin)
    }

    /// Retrieves blob data from Celestia at the specified height and commitment
    ///
    /// # Arguments
    ///
    /// * `inclusion_height` - Block height where the blob was included
    /// * `blob_commitment` - Commitment hash of the blob to retrieve
    ///
    /// # Returns
    ///
    /// Blob data as bytes or an error
    pub async fn get_blob(
        &self,
        inclusion_height: u64,
        blob_commitment: Commitment,
    ) -> Result<Vec<u8>, Box<dyn Error>> {
        let blob = self
            .celestia_client
            .get_blob(inclusion_height, &blob_commitment)
            .await?;
        Ok(blob.data)
    }

    /// Prepares the standard input for proof aggregation
    ///
    /// # Arguments
    ///
    /// * `inputs` - Vector of aggregation inputs containing proofs and verification keys
    ///
    /// # Returns
    ///
    /// Prepared SP1Stdin for aggregation or an error
    async fn get_aggregate_stdin(
        &self,
        inputs: Vec<AggregationInput>,
    ) -> Result<SP1Stdin, Box<dyn Error>> {
        assert!(inputs.len() > 1, "aggregation requires at least 2 proofs");

        // Create stdin for the aggregator
        let mut stdin = SP1Stdin::new();

        // Write the verification keys.
        let vkeys = inputs
            .iter()
            .map(|input| input.vk.hash_u32())
            .collect::<Vec<_>>();
        stdin.write::<Vec<[u32; 8]>>(&vkeys);

        // Write the public values.
        let public_values = inputs
            .iter()
            .map(|input| input.proof.public_values.to_vec())
            .collect::<Vec<_>>();
        stdin.write::<Vec<Vec<u8>>>(&public_values);

        // Write the proofs
        for input in &inputs {
            let SP1Proof::Compressed(ref proof) = input.proof.proof else {
                panic!()
            };
            stdin.write_proof(*proof.clone(), input.vk.vk.clone());
        }

        Ok(stdin)
    }

    /// Executes proof generation without creating a cryptographic proof
    ///
    /// # Arguments
    ///
    /// * `input` - Block prover input data
    ///
    /// # Returns
    ///
    /// Public values and execution report or an error
    pub async fn execute_generate_proof(
        &self,
        input: BlockProverInput,
    ) -> Result<(SP1PublicValues, ExecutionReport), Box<dyn Error>> {
        let stdin = self.get_stdin(input).await?;
        let (public_values, execution_report) = self
            .sp1_client
            .execute(self.prover_config.elf_bytes, &stdin)
            .run()
            .unwrap();

        Ok((public_values, execution_report))
    }

    /// Generates a cryptographic proof for a block
    ///
    /// # Arguments
    ///
    /// * `input` - Block prover input data
    ///
    /// # Returns
    ///
    /// Proof with public values and verification key or an error
    pub async fn generate_proof(
        &self,
        input: BlockProverInput,
    ) -> Result<(SP1ProofWithPublicValues, SP1VerifyingKey), Box<dyn Error>> {
        let (pk, vk) = self.sp1_client.setup(self.prover_config.elf_bytes);
        let stdin = self.get_stdin(input).await?;
        let proof = self.sp1_client.prove(&pk, &stdin).compressed().run()?;
        Ok((proof, vk))
    }

    /// Executes proof aggregation without creating a cryptographic proof
    ///
    /// # Arguments
    ///
    /// * `inputs` - Vector of aggregation inputs containing proofs and verification keys
    ///
    /// # Returns
    ///
    /// Public values and execution report or an error
    pub async fn execute_aggregate_proofs(
        &self,
        inputs: Vec<AggregationInput>,
    ) -> Result<(SP1PublicValues, ExecutionReport), Box<dyn Error>> {
        let stdin = self.get_aggregate_stdin(inputs).await?;
        let (public_values, execution_report) = self
            .sp1_client
            .execute(self.aggregator_config.elf_bytes, &stdin)
            .run()
            .unwrap();

        Ok((public_values, execution_report))
    }

    /// Aggregates multiple proofs into a single proof
    ///
    /// # Arguments
    ///
    /// * `inputs` - Vector of aggregation inputs containing proofs and verification keys
    ///
    /// # Returns
    ///
    /// Aggregation output containing the aggregated proof or an error
    pub async fn aggregate_proofs(
        &self,
        inputs: Vec<AggregationInput>,
    ) -> Result<AggregationOutput, Box<dyn Error>> {
        let stdin = self.get_aggregate_stdin(inputs).await?;
        let mode = std::env::var("SP1_PROVER").unwrap_or_else(|_| "cpu".to_string());

        if mode == "mock" {
            let mock_prover = sp1_sdk::CpuProver::mock();
            let (pk, _) = mock_prover.setup(self.aggregator_config.elf_bytes);
            let proof = mock_prover
                .prove(&pk, &stdin)
                .deferred_proof_verification(false)
                .run()?;

            Ok(AggregationOutput { proof })
        } else {
            let (pk, _) = self.sp1_client.setup(self.aggregator_config.elf_bytes);
            let proof: SP1ProofWithPublicValues = self.sp1_client.prove(&pk, &stdin).groth16().run()?;

            Ok(AggregationOutput { proof })
        }
    }

    /// Proves a range of blocks and aggregates their proofs
    ///
    /// # Arguments
    ///
    /// * `inputs` - Vector of block prover inputs for multiple blocks
    ///
    /// # Returns
    ///
    /// Aggregation output containing the aggregated proof or an error
    pub async fn prove_block_range(
        &self,
        inputs: Vec<BlockProverInput>,
    ) -> Result<AggregationOutput, Box<dyn Error>> {
        if inputs.len() < 2 {
            return Err("Must provide at least 2 proofs to aggregate".into());
        }

        // Generate proofs and collect verifying keys
        let mut agg_inputs = Vec::with_capacity(inputs.len());

        for input in inputs {
            let (proof, vk) = self.generate_proof(input).await?;

            agg_inputs.push(AggregationInput {
                proof,
                vk: vk.clone(),
            });
        }

        // Aggregate the proofs
        self.aggregate_proofs(agg_inputs).await
    }
}
