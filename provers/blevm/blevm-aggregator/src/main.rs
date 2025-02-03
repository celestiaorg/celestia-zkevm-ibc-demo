//! A SP1 program that takes as input public values from two blevm mock proofs. It then verifies
//! those mock proofs. Lastly, it verifies that the second proof is for an EVM block immediately
//! following the EVM block in proof one. It commits to the EVM header hashes from those two blocks.
#![no_main]
sp1_zkvm::entrypoint!(main);

mod buffer;
use buffer::Buffer;

use blevm_common::{BlevmAggOutput, BlevmOutput};
use sha2::{Digest, Sha256};

// Verification key of blevm-mock (Dec 22 2024)
// 0x001a3232969a5caac2de9a566ceee00641853a058b1ce1004ab4869f75a8dc59

const BLEVM_MOCK_VERIFICATION_KEY: [u32; 8] = [
    0x001a3232, 0x969a5caa, 0xc2de9a56, 0x6ceee006, 0x41853a05, 0x8b1ce100, 0x4ab4869f, 0x75a8dc59,
];

pub fn main() {
    let public_values1: Vec<u8> = sp1_zkvm::io::read();
    let public_values2: Vec<u8> = sp1_zkvm::io::read();

    let proof1_values_hash = Sha256::digest(&public_values1);
    let proof2_values_hash = Sha256::digest(&public_values2);

    sp1_zkvm::lib::verify::verify_sp1_proof(
        &BLEVM_MOCK_VERIFICATION_KEY,
        &proof1_values_hash.into(),
    );
    sp1_zkvm::lib::verify::verify_sp1_proof(
        &BLEVM_MOCK_VERIFICATION_KEY,
        &proof2_values_hash.into(),
    );

    let mut buffer1 = Buffer::from(&public_values1);
    let mut buffer2 = Buffer::from(&public_values2);

    let output1 = buffer1.read::<BlevmOutput>();
    let output2 = buffer2.read::<BlevmOutput>();

    if output1.header_hash != output2.prev_header_hash {
        panic!("header hash mismatch");
    }

    let agg_output = BlevmAggOutput {
        newest_header_hash: output2.header_hash,
        oldest_header_hash: output1.header_hash,
        celestia_header_hashes: vec![output1.celestia_header_hash, output2.celestia_header_hash],
        newest_state_root: output2.state_root,
        newest_height: output2.height,
    };

    sp1_zkvm::io::commit(&agg_output);
}
