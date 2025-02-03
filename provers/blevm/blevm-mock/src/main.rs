//! A SP1 program that commits the exact same output as blevm. This SP1 program should execute much
//! faster than blevm because it does not perform the same verification that blevm does. Note: this
//! should only be used for testing.
#![no_main]
sp1_zkvm::entrypoint!(main);

use blevm_common::BlevmOutput;

pub fn main() {
    let blob_commitment = sp1_zkvm::io::read::<Vec<u8>>();
    let header_hash = sp1_zkvm::io::read::<Vec<u8>>();
    let prev_header_hash = sp1_zkvm::io::read::<Vec<u8>>();
    let height = sp1_zkvm::io::read::<u64>();
    let gas_used = sp1_zkvm::io::read::<u64>();
    let beneficiary = sp1_zkvm::io::read::<Vec<u8>>();
    let state_root = sp1_zkvm::io::read::<Vec<u8>>();
    let celestia_header_hash = sp1_zkvm::io::read::<Vec<u8>>();

    let output = BlevmOutput {
        blob_commitment: blob_commitment.try_into().unwrap(),
        header_hash: header_hash.try_into().unwrap(),
        prev_header_hash: prev_header_hash.try_into().unwrap(),
        height,
        gas_used,
        beneficiary: beneficiary.try_into().unwrap(),
        state_root: state_root.try_into().unwrap(),
        celestia_header_hash: celestia_header_hash.try_into().unwrap(),
    };

    sp1_zkvm::io::commit(&output);
}
