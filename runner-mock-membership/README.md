# runner-mock-membership

This repo exists to help diagnose issues with the mock prover, and bench proof time for groth16.

## Running

1. Generate the ELF for mock-membership
  - `cd ../programs/sp1/mock-membership`
    `cargo prove build`
2. Run this
  - `cd ../../../runner-mock-membership`
  - `RUST_LOG=info cargo run --release`