//! A SP1 program that accepts several inputs and commits to several outputs. At a high-level it
//! accepts an EVM block and a few fields and proofs from a Celestia block. Then it verfifies that
//! the EVM block was included in the Celestia block. Lastly, it executes the EVM block and commits
//! the computed EVM header hash in the public outputs along with other metadata from the computed
//! EVM block.
#![no_main]

sp1_zkvm::entrypoint!(main);
use celestia_types::{blob::Blob, hash::Hash, AppVersion, ShareProof};
use eq_common::KeccakInclusionToDataRootProofInput;
use sha3::{Digest, Keccak256};

use blevm_common::BlevmOutput;
use nmt_rs::simple_merkle::tree::MerkleHash;
use nmt_rs::{simple_merkle::proof::Proof, TmSha2Hasher};
use rsp_client_executor::{executor::EthClientExecutor, io::EthClientExecutorInput};
use std::sync::Arc;
use tendermint::Hash as TmHash;

pub fn main() {
    println!("cycle-tracker-start: deserialize input");
    let input: KeccakInclusionToDataRootProofInput = sp1_zkvm::io::read();
    let data_root_as_hash = Hash::Sha256(input.data_root);
    let client_executor_input =
        bincode::deserialize::<EthClientExecutorInput>(&sp1_zkvm::io::read_vec()).unwrap();
    let celestia_header_hash: TmHash = sp1_zkvm::io::read();
    let data_hash_bytes: Vec<u8> = sp1_zkvm::io::read_vec();
    let proof_data_hash_to_celestia_hash: Proof<TmSha2Hasher> = sp1_zkvm::io::read();
    println!("cycle-tracker-end: deserialize input");

    println!("cycle-tracker-start: create blob");
    let blob =
        Blob::new(input.namespace_id, input.data, AppVersion::V3).expect("Failed creating blob");
    println!("cycle-tracker-end: create blob");

    println!("cycle-tracker-start: compute keccak hash");
    let computed_keccak_hash: [u8; 32] =
        Keccak256::new().chain_update(&blob.data).finalize().into();
    println!("cycle-tracker-end: compute keccak hash");

    println!("cycle-tracker-start: convert blob to shares");
    let rp = ShareProof {
        data: blob
            .to_shares()
            .expect("Failed to convert blob to shares")
            .into_iter()
            .map(|share| share.as_ref().try_into().unwrap())
            .collect(),
        namespace_id: input.namespace_id,
        share_proofs: input.share_proofs,
        row_proof: input.row_proof,
    };
    println!("cycle-tracker-end: convert blob to shares");

    // Verify that the data root goes into the Celestia block hash
    println!("cycle-tracker-start: verify data root");
    let hasher = TmSha2Hasher {};
    proof_data_hash_to_celestia_hash
        .verify_range(
            celestia_header_hash.as_bytes().try_into().unwrap(),
            &[hasher.hash_leaf(&data_hash_bytes)],
        )
        .unwrap();
    println!("cycle-tracker-end: verify data root");

    println!("cycle-tracker-start: verify proof");
    rp.verify(data_root_as_hash)
        .expect("Failed verifying proof");
    println!("cycle-tracker-end: verify proof");

    println!("cycle-tracker-start: check keccak hash");
    if computed_keccak_hash != input.keccak_hash {
        panic!("Computed keccak hash does not match input keccak hash");
    }
    println!("cycle-tracker-end: check keccak hash");

    // Execute the EVM block
    println!("cycle-tracker-start: executing EVM block");
    let executor = EthClientExecutor::eth(
        Arc::new((&client_executor_input.genesis).try_into().unwrap()),
        client_executor_input.custom_beneficiary,
    );
    let header = executor
        .execute(client_executor_input)
        .expect("failed to execute client");
    println!("cycle-tracker-end: executing EVM block");

    // Commit the new EVM header hash
    println!(
        "cycle-tracker-start: hashing the block header, and commiting fields as public values"
    );

    let output = BlevmOutput {
        blob_commitment: blob.commitment.into(),
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
}
