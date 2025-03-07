//! A SP1 program that commits the exact same output as blevm. This SP1 program should execute much
//! faster than blevm because it does not perform the same verification that blevm does. Note: this
//! should only be used for testing.
#![no_main]
sp1_zkvm::entrypoint!(main);

use blevm_common::BlevmOutput;

pub fn main() {
    // This is a mock proof so it hard-codes all the output values. Note: these values were sourced
    // from a valid execution of blevm.
    let output = BlevmOutput {
        blob_commitment: [
            196, 0, 0, 0, 0, 0, 0, 0, 121, 70, 207, 82, 142, 221, 116, 94, 251, 37, 32, 18, 70,
            230, 71, 213, 170, 202, 63, 181, 43, 240, 246, 6,
        ],
        header_hash: [
            182, 149, 180, 171, 11, 211, 63, 76, 133, 106, 134, 184, 20, 76, 104, 254, 40, 136, 41,
            140, 238, 199, 193, 86, 163, 56, 170, 193, 61, 146, 213, 227,
        ],
        prev_header_hash: [
            194, 70, 12, 164, 151, 147, 237, 105, 187, 154, 187, 153, 78, 140, 25, 59, 84, 254,
            152, 25, 224, 239, 83, 45, 145, 73, 226, 110, 100, 51, 95, 167,
        ],
        height: 18884864,
        gas_used: 14900876081506838043,
        beneficiary: [
            4, 26, 65, 0, 0, 0, 0, 0, 149, 34, 34, 144, 221, 114, 120, 170, 61, 221, 56, 156,
        ],
        state_root: [
            193, 225, 209, 101, 204, 75, 175, 229, 27, 56, 213, 58, 25, 68, 72, 76, 140, 126, 48,
            23, 127, 212, 219, 222, 63, 98, 45, 102, 165, 88, 255, 220,
        ],
        celestia_header_hash: [
            120, 107, 54, 46, 182, 50, 89, 93, 115, 224, 125, 214, 72, 215, 109, 67, 90, 48, 217,
            144, 215, 85, 206, 228, 192, 183, 123, 79, 244, 136, 195, 212,
        ],
    };
    let serialized_output = bincode::serialize(&output).unwrap();
    sp1_zkvm::io::commit(&serialized_output);
}
