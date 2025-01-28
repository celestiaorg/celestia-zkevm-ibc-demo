use bincode;
use celestia_rpc::{BlobClient, Client, HeaderClient};
use celestia_types::nmt::NamespacedHash;
use celestia_types::AppVersion;
use celestia_types::Blob;
use celestia_types::{
    nmt::{Namespace, NamespaceProof, NamespacedHashExt},
    ExtendedHeader,
};
use core::cmp::max;
use nmt_rs::{
    simple_merkle::{db::MemDb, proof::Proof, tree::MerkleTree},
    TmSha2Hasher,
};
use rsp_client_executor::io::ClientExecutorInput;
use sp1_sdk::{ProverClient, SP1Stdin};
use std::error::Error;
use tendermint_proto::{
    v0_37::{types::BlockId as RawBlockId, version::Consensus as RawConsensusVersion},
    Protobuf,
};

/// Configuration for the Celestia client
pub struct CelestiaConfig {
    pub node_url: String,
    pub auth_token: String,
}

/// Configuration for the proof generation
pub struct ProverConfig {
    pub elf_bytes: &'static [u8],
}

/// Input data for block proving
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
            .blob_get(height, self.namespace, blob.commitment.clone())
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
            .blob_get_proof(height, self.namespace, blob.commitment.clone())
            .await
            .map_err(|e| format!("Failed getting NMT proofs: {}", e))?)
    }
}

pub fn generate_header_proofs(
    header: &ExtendedHeader,
) -> Result<(Vec<u8>, Proof<TmSha2Hasher>), Box<dyn Error>> {
    let mut header_field_tree: MerkleTree<MemDb<[u8; 32]>, TmSha2Hasher> =
        MerkleTree::with_hasher(TmSha2Hasher::new());

    let field_bytes = prepare_header_fields(header);

    for leaf in field_bytes {
        header_field_tree.push_raw_leaf(&leaf);
    }

    let (data_hash_bytes, data_hash_proof) = header_field_tree.get_index_with_proof(6);

    // Verify the computed root matches the header hash
    assert_eq!(header.hash().as_ref(), header_field_tree.root());

    Ok((data_hash_bytes, data_hash_proof))
}

pub fn prepare_header_fields(header: &ExtendedHeader) -> Vec<Vec<u8>> {
    vec![
        Protobuf::<RawConsensusVersion>::encode_vec(header.header.version.clone()),
        header.header.chain_id.clone().encode_vec(),
        header.header.height.encode_vec(),
        header.header.time.encode_vec(),
        Protobuf::<RawBlockId>::encode_vec(header.header.last_block_id.unwrap_or_default()),
        header
            .header
            .last_commit_hash
            .unwrap_or_default()
            .encode_vec(),
        header.header.data_hash.unwrap_or_default().encode_vec(),
        header.header.validators_hash.encode_vec(),
        header.header.next_validators_hash.encode_vec(),
        header.header.consensus_hash.encode_vec(),
        header.header.app_hash.clone().encode_vec(),
        header
            .header
            .last_results_hash
            .unwrap_or_default()
            .encode_vec(),
        header.header.evidence_hash.unwrap_or_default().encode_vec(),
        header.header.proposer_address.encode_vec(),
    ]
}

pub fn generate_row_proofs(
    header: &ExtendedHeader,
    blob: &Blob,
    blob_index: u64,
) -> Result<(Proof<TmSha2Hasher>, Vec<NamespacedHash>), Box<dyn Error>> {
    let eds_row_roots = header.dah.row_roots();
    let eds_column_roots = header.dah.column_roots();
    let eds_size: u64 = eds_row_roots.len().try_into()?;
    let ods_size = eds_size / 2;

    let blob_size: u64 = max(1, blob.to_shares()?.len() as u64);
    let first_row_index: u64 = blob_index.div_ceil(eds_size) - 1;
    let ods_index = blob_index - (first_row_index * ods_size);
    let last_row_index: u64 = (ods_index + blob_size).div_ceil(ods_size) - 1;

    let mut row_root_tree: MerkleTree<MemDb<[u8; 32]>, TmSha2Hasher> =
        MerkleTree::with_hasher(TmSha2Hasher {});

    let leaves = eds_row_roots
        .iter()
        .chain(eds_column_roots.iter())
        .map(|root| root.to_array())
        .collect::<Vec<[u8; 90]>>();

    for root in &leaves {
        row_root_tree.push_raw_leaf(root);
    }

    let row_root_multiproof =
        row_root_tree.build_range_proof(first_row_index as usize..(last_row_index + 1) as usize);

    let selected_roots =
        eds_row_roots[first_row_index as usize..(last_row_index + 1) as usize].to_vec();

    Ok((row_root_multiproof, selected_roots))
}

/// Main prover service that coordinates the entire proving process
pub struct BlockProver {
    celestia_client: CelestiaClient,
    prover_config: ProverConfig,
}

impl BlockProver {
    pub fn new(celestia_client: CelestiaClient, prover_config: ProverConfig) -> Self {
        Self {
            celestia_client,
            prover_config,
        }
    }

    pub async fn generate_proof(&self, input: BlockProverInput) -> Result<Vec<u8>, Box<dyn Error>> {
        // Create blob from L2 block data
        let block: ClientExecutorInput = bincode::deserialize(&input.l2_block_data)?;
        let block_bytes = bincode::serialize(&block.current_block)?;
        let blob = Blob::new(
            self.celestia_client.namespace.clone(),
            block_bytes,
            AppVersion::V3,
        )?;

        // Get blob and header from Celestia
        let (blob_from_chain, header) = self
            .celestia_client
            .get_blob_and_header(input.block_height, &blob)
            .await?;

        // Generate all required proofs
        let (data_hash_bytes, data_hash_proof) = generate_header_proofs(&header)?;

        let (row_root_multiproof, selected_roots) =
            generate_row_proofs(&header, &blob_from_chain, blob_from_chain.index.unwrap())?;

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

        // Generate and return the proof
        let client = ProverClient::from_env();
        let (pk, _) = client.setup(self.prover_config.elf_bytes);
        let proof = client.prove(&pk, &stdin).core().run()?;

        bincode::serialize(&proof).map_err(|e| e.into())
    }
}
