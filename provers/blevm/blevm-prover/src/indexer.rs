use base64::{engine::general_purpose, Engine as _};
use reqwest::Error;
use serde::{Deserialize, Deserializer};
use std::time::Duration;

fn deserialize_base64<'de, D>(deserializer: D) -> Result<Vec<u8>, D::Error>
where
    D: Deserializer<'de>,
{
    let s = String::deserialize(deserializer)?;
    general_purpose::STANDARD
        .decode(&s)
        .map_err(serde::de::Error::custom)
}

/// Response structure for the inclusion height API
#[derive(Debug, Deserialize)]
struct InclusionHeightResponse {
    eth_block_number: u64,
    celestia_height: u64,
    #[serde(deserialize_with = "deserialize_base64")]
    blob_commitment: Vec<u8>,
}

/// Error returned by the get_inclusion_height function
#[derive(Debug, thiserror::Error)]
pub enum IndexerError {
    #[error("HTTP request error: {0}")]
    RequestError(#[from] Error),

    #[error("Block not found")]
    BlockNotFound,

    #[error("Server error: {0}")]
    ServerError(String),
}

/// Queries the indexer to get the Celestia block height and blob commitment for a specific EVM block
///
/// # Arguments
///
/// * `indexer_url` - The base URL of the indexer service
/// * `evm_block_height` - The Ethereum block height to query
///
/// # Returns
///
/// A tuple containing the Celestia block height and blob commitment, or an error
///
/// # Errors
///
/// Returns an `IndexerError` if:
/// - The HTTP request fails
/// - The block is not found
/// - The server returns an error
pub async fn get_inclusion_height(
    indexer_url: String,
    evm_block_height: u64,
) -> Result<(u64, Vec<u8>), IndexerError> {
    // Create a client with timeout
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(10))
        .build()?;

    // Build the request URL
    let url = format!(
        "{}/inclusion_height/{}",
        indexer_url.trim_end_matches('/'),
        evm_block_height
    );

    // Send the request
    println!("indexer: requesting url - {}", url.clone());
    let response = client.get(&url).send().await?;

    // Handle different status codes
    match response.status() {
        reqwest::StatusCode::OK => {
            // Parse the response body
            let data: InclusionHeightResponse = response.json().await?;
            // Sanity check
            assert!(data.eth_block_number == evm_block_height);
            Ok((data.celestia_height, data.blob_commitment))
        }
        reqwest::StatusCode::NOT_FOUND => Err(IndexerError::BlockNotFound),
        status => {
            // For other error cases, try to get error message
            let error_text = response.text().await.unwrap_or_else(|_| status.to_string());
            Err(IndexerError::ServerError(error_text))
        }
    }
}
