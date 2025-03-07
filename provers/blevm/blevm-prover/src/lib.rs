mod proofs;

use celestia_rpc::{BlobClient, Client, HeaderClient};
use celestia_types::AppVersion;
use celestia_types::Blob;
use celestia_types::{
    nmt::{Namespace, NamespaceProof},
    ExtendedHeader,
};
use rsp_client_executor::io::ClientExecutorInput;
use serde::{Deserialize, Serialize};
use sp1_sdk::{
    ExecutionReport, ProverClient, SP1ProofWithPublicValues, SP1PublicValues, SP1Stdin,
    SP1VerifyingKey, SP1Proof
};
use std::error::Error;

/// Configuration for the Celestia client
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
#[derive(Serialize, Deserialize)]
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
    pub block_height: u64,
    pub l2_block_data: Vec<u8>,
}

/// Handles interaction with Celestia network
pub struct CelestiaClient {
    client: Client,
    namespace: Namespace,
}

impl CelestiaClient {
    pub async fn new(config: CelestiaConfig, namespace: Namespace) -> Result<Self, Box<dyn Error>> {
        let client = Client::new(&config.node_url, Some(&config.auth_token))
            .await
            .map_err(|e| format!("Failed creating RPC client: {}", e))?;

        Ok(Self { client, namespace })
    }

    pub async fn get_blob_and_header(
        &self,
        height: u64,
        blob: &Blob,
    ) -> Result<(Blob, ExtendedHeader), Box<dyn Error>> {
        let blob_from_chain = self
            .client
            .blob_get(height, self.namespace, blob.commitment)
            .await
            .map_err(|e| format!("Failed getting blob: {}", e))?;

        let header = self
            .client
            .header_get_by_height(height)
            .await
            .map_err(|e| format!("Failed getting header: {}", e))?;

        Ok((blob_from_chain, header))
    }

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
}

impl BlockProver {
    pub fn new(
        celestia_client: CelestiaClient,
        prover_config: ProverConfig,
        aggregator_config: AggregatorConfig,
    ) -> Self {
        Self {
            celestia_client,
            prover_config,
            aggregator_config,
        }
    }

    async fn get_stdin(&self, input: BlockProverInput) -> Result<SP1Stdin, Box<dyn Error>> {
        // Create blob from L2 block data
        let block: ClientExecutorInput = bincode::deserialize(&input.l2_block_data)?;
        let block_bytes = bincode::serialize(&block.current_block)?;
        let blob = Blob::new(self.celestia_client.namespace, block_bytes, AppVersion::V3)?;

        // Get blob and header from Celestia
        let (blob_from_chain, header) = self
            .celestia_client
            .get_blob_and_header(input.block_height, &blob)
            .await?;

        // Generate all required proofs
        let (data_hash_bytes, data_hash_proof) = proofs::generate_header_proofs(&header)?;

        let (row_root_multiproof, selected_roots) =
            proofs::generate_row_proofs(&header, &blob_from_chain, blob_from_chain.index.unwrap())?;

        let nmt_multiproofs = self
            .celestia_client
            .get_nmt_proofs(input.block_height, &blob)
            .await?;

        // Prepare stdin for the prover
        let mut stdin = SP1Stdin::new();
        stdin.write(&block);
        stdin.write(&self.celestia_client.namespace);
        stdin.write(&header.header.hash());
        stdin.write_vec(data_hash_bytes);
        stdin.write(&data_hash_proof);
        stdin.write(&row_root_multiproof);
        stdin.write(&nmt_multiproofs);
        stdin.write(&selected_roots);
        Ok(stdin)
    }

    async fn get_aggregate_stdin(
        &self,
        inputs: Vec<AggregationInput>,
    ) -> Result<SP1Stdin, Box<dyn Error>> {
        if inputs.len() < 2 {
            return Err("Must provide at least 2 proofs to aggregate".into());
        }

        // Create stdin for the aggregator
        let mut stdin = SP1Stdin::new();

        // Write number of proofs
        stdin.write(&inputs.len());

        // Write all verification keys first
        for input in &inputs {
            stdin.write(&input.vk);
        }

        // Then write all public values
        for input in &inputs {
            stdin.write_vec(input.proof.public_values.to_vec());
        }

        // Write the proofs
        for input in &inputs {
            let SP1Proof::Compressed(ref proof) = input.proof.proof else { panic!() };
            stdin.write_proof(*proof.clone(), input.vk.vk.clone());
        }

        Ok(stdin)
    }

    pub async fn execute_generate_proof(
        &self,
        input: BlockProverInput,
    ) -> Result<(SP1PublicValues, ExecutionReport), Box<dyn Error>> {
        let client: sp1_sdk::EnvProver = ProverClient::from_env();
        let stdin = self.get_stdin(input).await?;
        let (public_values, execution_report) = client
            .execute(self.prover_config.elf_bytes, &stdin)
            .run()
            .unwrap();

        Ok((public_values, execution_report))
    }

    pub async fn generate_proof(
        &self,
        input: BlockProverInput,
    ) -> Result<(SP1ProofWithPublicValues, SP1VerifyingKey), Box<dyn Error>> {
        // Generate and return the proof
        let client: sp1_sdk::EnvProver = ProverClient::from_env();
        let (pk, vk) = client.setup(self.prover_config.elf_bytes);
        let stdin = self.get_stdin(input).await?;
        let proof = client.prove(&pk, &stdin).compressed().run()?;
        Ok((proof, vk))
    }

    pub async fn execute_aggregate_proofs(
        &self,
        inputs: Vec<AggregationInput>,
    ) -> Result<(SP1PublicValues, ExecutionReport), Box<dyn Error>> {
        let client: sp1_sdk::EnvProver = ProverClient::from_env();
        let stdin = self.get_aggregate_stdin(inputs).await?;
        let (public_values, execution_report) = client
            .execute(self.aggregator_config.elf_bytes, &stdin)
            .run()
            .unwrap();

        Ok((public_values, execution_report))
    }

    /// Aggregates multiple proofs into a single proof
    pub async fn aggregate_proofs(
        &self,
        inputs: Vec<AggregationInput>,
    ) -> Result<AggregationOutput, Box<dyn Error>> {
        let stdin = self.get_aggregate_stdin(inputs).await?;
        let client: sp1_sdk::EnvProver = ProverClient::from_env();
        // Generate the aggregated proof
        let (pk, _) = client.setup(self.aggregator_config.elf_bytes);
        let proof = client.prove(&pk, &stdin).groth16().run()?;

        Ok(AggregationOutput { proof })
    }

    /// Proves a range of blocks and aggregates their proofs
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
