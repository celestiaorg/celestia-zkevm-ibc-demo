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
    pub newest_header_hash: [u8; 32],
    pub oldest_header_hash: [u8; 32],
    pub celestia_header_hashes: Vec<[u8; 32]>,
    pub newest_state_root: [u8; 32],
    pub newest_height: u64,
}
