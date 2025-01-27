# blevm

blEVM is a service that creates zero-knowledge proofs of EVM state transitions.

## Contributing

### Prerequisites

1. Install Rust > 1.81.0

### Usage

```shell
# Run a DA light node on Mainnet. Generate an auth token and export it as an env variable.
export CELESTIA_NODE_AUTH_TOKEN=$(celestia light auth admin)

# Change to the correct directory
cd celestia-zkevm-ibc-demo/provers/blevm/script

# Run the script
cargo run main.rs
```
