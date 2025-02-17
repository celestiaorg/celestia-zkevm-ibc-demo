use celestia_types::nmt::NamespacedHash;
use celestia_types::Blob;
use celestia_types::{nmt::NamespacedHashExt, ExtendedHeader};
use core::cmp::max;
use nmt_rs::{
    simple_merkle::{db::MemDb, proof::Proof, tree::MerkleTree},
    TmSha2Hasher,
};
use std::error::Error;
use tendermint_proto::{
    v0_37::{types::BlockId as RawBlockId, version::Consensus as RawConsensusVersion},
    Protobuf,
};

/// generate_header_proofs takes an extender header and creates a Merkle tree from its fields. Then
/// it generates a Merkle proof for the DataHash in that extended header.
pub fn generate_header_proofs(
    header: &ExtendedHeader,
) -> Result<(Vec<u8>, Proof<TmSha2Hasher>), Box<dyn Error>> {
    let mut header_field_tree: MerkleTree<MemDb<[u8; 32]>, TmSha2Hasher> =
        MerkleTree::with_hasher(TmSha2Hasher::new());

    let field_bytes = prepare_header_fields(header);

    for leaf in field_bytes {
        header_field_tree.push_raw_leaf(&leaf);
    }

    // The data_hash is the leaf at index 6 in the tree.
    let (data_hash_bytes, data_hash_proof) = header_field_tree.get_index_with_proof(6);

    // Verify the computed root matches the header hash
    assert_eq!(header.hash().as_ref(), header_field_tree.root());

    Ok((data_hash_bytes, data_hash_proof))
}

/// prepare_header_fields returns a vector with all the fields in a Tendermint header.
/// See https://github.com/cometbft/cometbft/blob/972fa8038b57cc2152cb67144869ccd604526550/spec/core/data_structures.md?plain=1#L130-L143
pub fn prepare_header_fields(header: &ExtendedHeader) -> Vec<Vec<u8>> {
    vec![
        Protobuf::<RawConsensusVersion>::encode_vec(header.header.version),
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
