use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
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

#[derive(Serialize, Deserialize)]
pub struct BlevmAggOutput {
    // newest_header_hash is the last block's hash on the EVM roll-up.
    // TODO: this may be removable.
    pub newest_header_hash: [u8; 32],
    // oldest_header_hash is the earliest block's hash on the EVM roll-up.
    // TODO: this may be removable.
    pub oldest_header_hash: [u8; 32],
    // celestia_header_hashes is the range of Celestia blocks that include all
    // of the blob data the EVM roll-up has posted from oldest_header_hash to
    // newest_header_hash.
    pub celestia_header_hashes: Vec<[u8; 32]>, // provided by Celestia state machine (eventually x/header)
    // newest_state_root is the computed state root of the EVM roll-up after
    // processing blocks from oldest_header_hash to newest_header_hash.
    pub newest_state_root: [u8; 32],
    // newest_height is the most recent block number of the EVM roll-up.
    // TODO: this may be removable.
    pub newest_height: u64,
}
