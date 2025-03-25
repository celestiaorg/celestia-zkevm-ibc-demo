use alloy_provider::network::Ethereum;
use eyre::Context;
use reth_chainspec::ChainSpec;
use rsp_host_executor::EthHostExecutor;
use rsp_primitives::genesis::Genesis;
use rsp_rpc_db::RpcDb;
use std::sync::Arc;

/// Generate client input for the specified block
async fn generate_client_input(
    provider: impl alloy_provider::Provider<Ethereum> + Clone,
    block_number: u64,
    genesis: &Genesis,
    custom_beneficiary: Option<String>,
    opcode_tracking: bool,
) -> eyre::Result<Vec<u8>> {
    let chain_spec: Arc<ChainSpec> = Arc::new(genesis.try_into().unwrap());

    let custom_beneficiary = custom_beneficiary
        .as_deref()
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
