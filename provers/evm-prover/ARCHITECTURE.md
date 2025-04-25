# EVM Prover

The EVM Prover implements the Prover service for EVM rollups, creating zero-knowledge range proofs for state membership and state transitions. It uses the following SP1 programs to generate these range proofs.

## SP1 Programs

### blevm

The `blevm` SP1 program generates proofs of data inclusion and execution for single EVM blocks.

#### Inputs:
- `KeccakInclusionToDataRootProofInput`: Data for inclusion proof
- `EthClientExecutorInput`: Data for EVM execution
- `celestia_header_hash`: Celestia block header hash
- `data_hash_bytes`: Hash of the block data
- `proof_data_hash_to_celestia_hash`: Merkle proof connecting data hash to Celestia hash

#### Output:
```rust
pub struct BlevmOutput {
    pub blob_commitment: [u8; 32],
    pub header_hash: [u8; 32],
    pub prev_header_hash: [u8; 32],
    pub height: u64,
    pub gas_used: u64,
    pub beneficiary: [u8; 20],
    pub state_root: [u8; 32],
    pub celestia_header_hash: [u8; 32],
}
```

### blevm-aggregator

The `blevm-aggregator` SP1 program aggregates multiple block proofs to create a proof over a range of blocks.

#### Inputs:
- `vkeys`: Verification keys for the proofs to aggregate
- `public_values`: Public values from the proofs to aggregate
- Proofs (automatically input)

#### Output:
```rust
pub struct BlevmAggOutput {
    pub newest_header_hash: [u8; 32],
    pub oldest_header_hash: [u8; 32],
    pub celestia_header_hashes: Vec<[u8; 32]>,
    pub newest_state_root: [u8; 32],
    pub newest_height: u64,
}
```
## Rollkit Indexer

The current architecture uses an indexer for Rollkit which:
- Maps EVM block heights to their inclusion heights in Celestia
- Stores pointers i.e inclusion height, blob commitment to data submitted to Celestia
- Provides these as inputs for proving inclusion

This mapping is used by the EVM Prover to locate the Celestia inclusion block when generating proofs.

## Current Limitations and Future Work

### Current Limitations

1. **Matching Inclusion and Execution Data**: Matching inclusion data to execution data is not yet implemented.
2. **Sequencer Signature Verification**: Verification of sequencer signatures is not yet implementated.

### Future Improvements

1. **Modular Proofs**: In the future, we should implement modular proofs that can:
   - Create inclusion, execution, and signature verification proofs in parallel
   - Verify proofs over a requested range in parallel
   - Improve throughput and reduce latency for proof generation

2. **Remove Indexer**: The indexer is a stop-gap solution, in future Rollkit versions we can lookup the inclusion block more efficiently.

3. **Optimized Proof Aggregation**: Implement logarithmic algorithm for proof aggregation to reduce computational overhead.
