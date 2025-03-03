//! A program that verifies the membership or non-membership of a value in a commitment root.

#![deny(missing_docs, clippy::nursery, clippy::pedantic, warnings)]
#![allow(clippy::no_mangle_with_rust_abi)]
// These two lines are necessary for the program to properly compile.
//
// Under the hood, we wrap your main function with some extra code so that it behaves properly
// inside the zkVM.
#![no_main]
sp1_zkvm::entrypoint!(main);

use alloy_sol_types::SolValue;

use membership_fast::membership;

/// The main function of the program.
///
/// # Panics
/// Panics if the verification fails.
/// The main function of the program, simplified to just commit inputs.
///
/// # Panics
/// Panics if the input reading fails.
pub fn main() {
    let encoded_1 = sp1_zkvm::io::read_vec();
    let app_hash: [u8; 32] = encoded_1.try_into().unwrap();

    let request_len = sp1_zkvm::io::read_vec()[0];
    assert!(request_len != 0);

    let request_iter = (0..request_len).map(|_| {
        let path = sp1_zkvm::io::read_vec();
        let value = sp1_zkvm::io::read_vec();

        // Wrap the `path` in another Vec to satisfy the expected type `(Vec<Vec<u8>>, Vec<u8>)`
        (vec![path], value)
    });

    let output = membership(app_hash, request_iter);

    sp1_zkvm::io::commit_slice(&output.abi_encode());
}
