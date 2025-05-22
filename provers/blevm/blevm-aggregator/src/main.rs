//! A SP1 program that takes as input N verification keys and N public values from N blevm proofs.
//! It then verifies those proofs. It verifies that each proof is for an EVM block immediately
//! following the previous block. It commits to the EVM header hashes from the first and last
//! blocks. Note that the proofs must be in order of increasing block height.
#![no_main]
sp1_zkvm::entrypoint!(main);

mod buffer;
use blevm_common::{BlevmAggOutput, BlevmOutput};
use buffer::Buffer;
use sha2::{Digest, Sha256};

pub fn main() {
    // Read the verification keys.
    let vkeys = sp1_zkvm::io::read::<Vec<[u32; 8]>>();

    // Read the public values.
    let public_values = sp1_zkvm::io::read::<Vec<Vec<u8>>>();

    // Verify the proofs.
    assert_eq!(vkeys.len(), public_values.len());
    for i in 0..vkeys.len() {
        let vkey = &vkeys[i];
        let public_values = &public_values[i];
        let public_values_digest = Sha256::digest(public_values);
        sp1_zkvm::lib::verify::verify_sp1_proof(vkey, &public_values_digest.into());
    }

    // Parse all outputs
    let mut outputs: Vec<BlevmOutput> = vec![];
    for values in &public_values {
        let mut buffer = Buffer::from(values);
        outputs.push(buffer.read::<BlevmOutput>());
    }

    // // Verify adjacent headers
    for i in 1..outputs.len() {
        if outputs[i].prev_header_hash != outputs[i - 1].header_hash {
            panic!("header hash mismatch at position {}", i);
        }
    }

    // Collect all Celestia header hashes
    let celestia_header_hashes: Vec<_> = outputs
        .iter()
        .map(|output| output.celestia_header_hash)
        .collect();

    // Create aggregate output using first and last blocks
    let agg_output = BlevmAggOutput {
        newest_header_hash: outputs[vkeys.len() - 1].header_hash,
        oldest_header_hash: outputs[0].header_hash,
        celestia_header_hashes,
        newest_state_root: outputs[vkeys.len() - 1].state_root,
        newest_height: outputs[vkeys.len() - 1].height,
    };
    // Print all the agg output values
    println!("agg_output: {:?}", agg_output);
    // print them separately
    println!("newest_header_hash: {:?}", agg_output.newest_header_hash);
    println!("oldest_header_hash: {:?}", agg_output.oldest_header_hash);
    println!("celestia_header_hashes: {:?}", agg_output.celestia_header_hashes);
    println!("newest_state_root: {:?}", agg_output.newest_state_root);
    println!("newest_height: {:?}", agg_output.newest_height);

    sp1_zkvm::io::commit(&agg_output);
}
