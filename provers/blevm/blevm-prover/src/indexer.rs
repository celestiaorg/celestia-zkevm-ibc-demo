use reqwest::Error;
use serde::Deserialize;
use std::time::Duration;

/// Response structure for the inclusion height API
#[derive(Debug, Deserialize)]
struct InclusionHeightResponse {
    eth_block_number: u64,
    celestia_height: u64,
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

pub async fn get_inclusion_height(
    indexer_url: String,
    evm_block_height: u64,
) -> Result<u64, IndexerError> {
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
    let response = client.get(&url).send().await?;

    // Handle different status codes
    match response.status() {
        reqwest::StatusCode::OK => {
            // Parse the response body
            let data: InclusionHeightResponse = response.json().await?;
            Ok(data.celestia_height)
        }
        reqwest::StatusCode::NOT_FOUND => Err(IndexerError::BlockNotFound),
        status => {
            // For other error cases, try to get error message
            let error_text = response.text().await.unwrap_or_else(|_| status.to_string());
            Err(IndexerError::ServerError(error_text))
        }
    }
}
