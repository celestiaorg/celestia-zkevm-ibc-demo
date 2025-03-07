//! A SP1 program that takes as input N verification keys and N public values from N blevm proofs.
//! It then verifies those proofs. It verifies that each proof is for an EVM block immediately
//! following the previous block. It commits to the EVM header hashes from the first and last
//! blocks.
#![no_main]
sp1_zkvm::entrypoint!(main);

mod buffer;
use blevm_common::{BlevmAggOutput, BlevmOutput};
use buffer::Buffer;
use sha2::{Digest, Sha256};

pub fn main() {
    // Read the number of proofs
    let n: usize = sp1_zkvm::io::read();

    if n < 2 {
        panic!("must provide at least 2 proofs");
    }

    // Read all verification keys first
    let mut verification_keys: Vec<[u32; 8]> = Vec::with_capacity(n);
    for _ in 0..n {
        verification_keys.push(sp1_zkvm::io::read());
    }

    // Read all public values
    let mut public_values: Vec<Vec<u8>> = Vec::with_capacity(n);
    for _ in 0..n {
        public_values.push(sp1_zkvm::io::read());
    }

    // Verify all proofs using their respective verification keys
    for i in 0..n {
        let vkey = &verification_keys[i];
        let public_values = &public_values[i];
        let public_values_digest = Sha256::digest(public_values);
        sp1_zkvm::lib::verify::verify_sp1_proof(vkey, &public_values_digest.into());
    }

    // Parse all outputs
    let mut outputs: Vec<BlevmOutput> = Vec::with_capacity(n);
    for values in &public_values {
        let mut buffer = Buffer::from(values);
        outputs.push(buffer.read::<BlevmOutput>());
    }

    // Verify block sequence
    for i in 1..n {
        if outputs[i - 1].header_hash != outputs[i].prev_header_hash {
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
        newest_header_hash: outputs[n - 1].header_hash,
        oldest_header_hash: outputs[0].header_hash,
        celestia_header_hashes,
        newest_state_root: outputs[n - 1].state_root,
        newest_height: outputs[n - 1].height,
    };

    sp1_zkvm::io::commit(&agg_output);
}
