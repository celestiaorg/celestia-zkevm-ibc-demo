use alloy_provider::ProviderBuilder;
use eyre::Context;
use reth_chainspec::ChainSpec;
use rsp_host_executor::EthHostExecutor;
use rsp_primitives::genesis::Genesis;
use rsp_rpc_db::RpcDb;
use std::sync::Arc;

/// Generates the serialized client input for the execution of a specific block.
///
/// # Arguments
///
/// * `evm_rpc_url` - URL of the EVM RPC endpoint to connect to
/// * `block_number` - The block number to execute
/// * `genesis` - Genesis configuration for the chain
/// * `custom_beneficiary` - Optional custom beneficiary address for block rewards
/// * `opcode_tracking` - Whether to enable opcode tracking during execution
///
/// # Returns
///
/// Serialized client input as a vector of bytes, or an error if execution fails.
///
/// # Errors
///
/// Returns an error if:
/// - The custom beneficiary address cannot be parsed
/// - Block execution fails
/// - Client input serialization fails
pub async fn generate_client_input(
    evm_rpc_url: String,
    block_number: u64,
    genesis: &Genesis,
    custom_beneficiary: Option<&String>,
    opcode_tracking: bool,
) -> eyre::Result<Vec<u8>> {
    let provider = ProviderBuilder::new().on_http(evm_rpc_url.parse().unwrap());

    let chain_spec: Arc<ChainSpec> = Arc::new(genesis.try_into().unwrap());

    let custom_beneficiary = custom_beneficiary
        .map(|addr| addr.parse())
        .transpose()
        .wrap_err("Failed to parse custom beneficiary address")?;

    // Create host executor
    let host_executor = EthHostExecutor::eth(chain_spec.clone(), custom_beneficiary);

    // Create RPC DB with oldest ancestor block
    let rpc_db = RpcDb::new(provider.clone(), block_number - 1);

    // Execute block to generate client input
    let client_input = host_executor
        .execute(
            block_number,
            &rpc_db,
            &provider,
            genesis.clone(),
            custom_beneficiary,
            opcode_tracking,
        )
        .await
        .wrap_err_with(|| format!("Failed to execute block {}", block_number))?;

    // Serialize client input to bincode
    let encoded = bincode::serialize(&client_input)
        .wrap_err("Failed to serialize client input to bincode")?;

    Ok(encoded)
}
