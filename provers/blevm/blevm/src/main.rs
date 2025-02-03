//! A SP1 program that accepts several inputs and commits to several outputs. At a high-level it
//! accepts an EVM block and a few fields and proofs from a Celestia block. Then it verfifies that
//! the EVM block was included in the Celestia block. Lastly, it executes the EVM block and commits
//! the computed EVM header hash in the public outputs along with other metadata from the computed
//! EVM block.
#![no_main]
sp1_zkvm::entrypoint!(main);

use blevm_common::BlevmOutput;
use celestia_types::nmt::{NamespaceProof, NamespacedHashExt};
use celestia_types::{nmt::Namespace, AppVersion, Blob};
use nmt_rs::simple_merkle::tree::MerkleHash;
use nmt_rs::{simple_merkle::proof::Proof, NamespacedHash, TmSha2Hasher};
use rsp_client_executor::{io::ClientExecutorInput, ClientExecutor, EthereumVariant};
use tendermint::Hash as TmHash;
use tendermint_proto::Protobuf;

pub fn main() {
    println!("cycle-tracker-start: cloning and deserializing inputs");
    let input: ClientExecutorInput = sp1_zkvm::io::read();
    // namespace is the namespace on Celestia that includes the roll-up block data.
    let namespace: Namespace = sp1_zkvm::io::read();
    let celestia_header_hash: TmHash = sp1_zkvm::io::read();
    let data_hash_bytes: Vec<u8> = sp1_zkvm::io::read_vec();
    // data_hash is the Merkle root of the hash of transactions in a Celestia block.
    // Ref: https://github.com/cometbft/cometbft/blob/972fa8038b57cc2152cb67144869ccd604526550/spec/core/data_structures.md?plain=1#L136
    let data_hash: TmHash = TmHash::decode_vec(&data_hash_bytes).unwrap();
    // data_hash_proof is a Merkle proof that data_hash is a member of the Merkle tree with root
    // celestia_header_hash.
    let data_hash_proof: Proof<TmSha2Hasher> = sp1_zkvm::io::read();
    let row_root_multiproof: Proof<TmSha2Hasher> = sp1_zkvm::io::read();
    let nmt_multiproofs: Vec<NamespaceProof> = sp1_zkvm::io::read();
    let row_roots: Vec<NamespacedHash<29>> = sp1_zkvm::io::read();

    let block = input.current_block.clone();
    println!("cycle-tracker-end: cloning and deserializing inputs");

    // Verify that the data hash is a member of the Merkle tree with root celestia_header_hash. In
    // other words, verify that the data hash is present in the Celestia header.
    println!("cycle-tracker-start: verify data hash");
    let hasher = TmSha2Hasher {};
    data_hash_proof
        .verify_range(
            celestia_header_hash.as_bytes().try_into().unwrap(),
            &[hasher.hash_leaf(&data_hash_bytes)],
        )
        .unwrap();
    println!("cycle-tracker-end: verify data hash");

    println!("cycle-tracker-start: serializing EVM block");
    let block_bytes = bincode::serialize(&block).unwrap();
    println!("cycle-tracker-end: serializing EVM block");

    println!("cycle-tracker-start: creating Blob");
    // Convert the EVM block into a Celestia blob.
    let blob = Blob::new(namespace, block_bytes, AppVersion::V3).unwrap();
    println!("Blob commitment: {}", hex::encode(blob.commitment.0));
    println!("cycle-tracker-end: creating Blob");

    println!("cycle-tracker-start: blob to shares");
    // Convert the blob into Celestia shares.
    let shares = blob.to_shares().unwrap();
    println!("cycle-tracker-end: blob to shares");

    // Verify NMT multiproofs of blob shares into row roots
    println!("cycle-tracker-start: verify NMT multiproofs of blob shares into row roots");
    let mut start = 0;
    for i in 0..nmt_multiproofs.len() {
        let proof = &nmt_multiproofs[i];
        let end = start + (proof.end_idx() as usize - proof.start_idx() as usize);
        proof
            .verify_range(&row_roots[i], &shares[start..end], namespace.into())
            .expect("NMT multiproof into row root failed verification"); // Panicking should prevent an invalid proof from being generated
        start = end;
    }
    println!("cycle-tracker-end: verify NMT multiproofs of blob shares into row roots");

    // Verify row root inclusion into data root
    println!("cycle-tracker-start: verify row root inclusion into data root");
    let tm_hasher = TmSha2Hasher {};
    let blob_row_root_hashes: Vec<[u8; 32]> = row_roots
        .iter()
        .map(|root| tm_hasher.hash_leaf(&root.to_array()))
        .collect();
    let _result = row_root_multiproof.verify_range(
        data_hash.as_bytes().try_into().unwrap(),
        &blob_row_root_hashes,
    );
    println!("cycle-tracker-end: verify row root inclusion into data root");

    // Execute the EVM block
    println!("cycle-tracker-start: executing EVM block");
    let executor = ClientExecutor;
    let header = executor.execute::<EthereumVariant>(input).unwrap(); // panicking should prevent a proof of invalid execution from being generated
    println!("cycle-tracker-end: executing EVM block");

    // Commit the new EVM header hash
    println!(
        "cycle-tracker-start: hashing the block header, and commiting fields as public values"
    );

    let output = BlevmOutput {
        blob_commitment: blob.commitment.0,
        // header_hash is for the EVM executed block
        header_hash: header.hash_slow().into(),
        prev_header_hash: header.parent_hash.into(),
        height: header.number,
        gas_used: header.gas_used,
        beneficiary: header.beneficiary.into(),
        state_root: header.state_root.into(),
        celestia_header_hash: celestia_header_hash.as_bytes().try_into().unwrap(),
    };
    sp1_zkvm::io::commit(&output);

    println!(
        "cycle-tracker-end: hashing the block header, and commiting its fields as public values"
    );
}
