# blevm

blevm is a service that creates zero-knowledge proofs of EVM state transitions.

## Contributing

### Prerequisites

1. Install Rust > 1.81.0
1. Create and populate the `.env` file

    ```shell
    cp .env.example .env
    # Modify the .env file and set `SP1_PROVER=network` and `NETWORK_PRIVATE_KEY="PRIVATE_KEY"` to the SP1 prover network private key from Celestia 1Password.
    ```

### Usage

```shell
# Initialize a Celestia light node
celestia light init
# We need to sync from 2988870 onwards because script/main.rs queries that height.
# Set the DASer.SampleFrom SampleFrom = 2988870
vim ~/.celestia-light/config.toml
# Set the trusted hash to the last block hash.
# curl -s "https://rpc.celestia.pops.one/block?height=2988870" | jq -r '.result.block.header.last_block_id.hash'
# FFF21255D1CE0EECB8B491173F547A42665C3C7468C9B8855F7BC91E69B19BC3
export TRUSTED_HASH=FFF21255D1CE0EECB8B491173F547A42665C3C7468C9B8855F7BC91E69B19BC3
# Run a DA light node on Mainnet.
celestia light start --core.ip rpc.celestia.pops.one --p2p.network celestia --headers.trusted-hash $TRUSTED_HASH
# Generate an auth token and export it as an env variable.
export CELESTIA_NODE_AUTH_TOKEN=$(celestia light auth admin)
# Export namespace that was used to post an EVM block
export CELESTIA_NAMESPACE=0f0f0f0f0f0f0f0f0f0f

# Change to the correct directory
cd celestia-zkevm-ibc-demo/provers/blevm/script
# Run the script
cargo run
```
