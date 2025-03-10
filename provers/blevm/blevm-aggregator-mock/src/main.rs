#![no_main]
sp1_zkvm::entrypoint!(main);

mod buffer;
use blevm_common::{BlevmAggOutput, BlevmOutput};
use buffer::Buffer;

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

    // Parse all outputs and collect Celestia header hashes
    let mut outputs: Vec<BlevmOutput> = Vec::with_capacity(n);
    let mut celestia_header_hashes: Vec<[u8; 32]> = Vec::with_capacity(n);
    for values in &public_values {
        let mut buffer = Buffer::from(values);
        let output: BlevmOutput = buffer.read();
        celestia_header_hashes.push(output.celestia_header_hash);
        outputs.push(output);
    }

    // Create aggregate output using first and last blocks
    let agg_output = BlevmAggOutput {
        newest_header_hash: outputs[n - 1].header_hash,
        oldest_header_hash: outputs[0].header_hash,
        celestia_header_hashes,
        newest_state_root: outputs[n - 1].state_root,
        newest_height: outputs[n - 1].height,
    };

    // Commit the aggregate output
    sp1_zkvm::io::commit(&agg_output);
}
