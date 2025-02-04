# blevm

blevm is a service that creates zero-knowledge proofs of EVM state transitions.

## Project layout

This workspace contains multiple crates:

- `blevm`: SP1 program that verifies an EVM block was included in a Celestia data square.
- `blevm-mock`: SP1 program that acts as a mock version of `blevm`. It should execute faster than `blevm` because it skips verifying any inputs or outputs.
- `blevm-aggregator`: SP1 program that takes as input the public values from two `blevm` proofs. It verifies the proofs and ensures they are for monotonically increasing EVM blocks.
- `blevm-prover`: library that exposes a `BlockProver` which can generate proofs. The proofs can either be `blevm` proofs or `blevm-mock` proofs depending on the `elf_bytes` used.
- `common`: library with common struct definitions
- `script`: binary that generates a blevm proof for an EVM roll-up block that was posted to Celestia mainnet.

## Contributing

See <https://docs.succinct.xyz/docs/introduction>

### Prerequisites

1. Install Rust > 1.81.0
1. Create and populate the `.env` file

    ```shell
    cp .env.example .env
    # Modify the .env file and set `SP1_PROVER=network` and `NETWORK_PRIVATE_KEY="PRIVATE_KEY"` to the SP1 prover network private key from Celestia 1Password.
    ```

### Usage

The `script` binary will generate an SP1 proof but it depends on a DA node. You can either run a light node locally or proxy to a full node running in Lunaroasis.

1. [Optional] Run a light node locally

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
    ```

2. [Optional] Proxy to a full node running in Lunaroasis.
    1. Open Lens.
    1. Connect to lunaroasis.
    1. Navigate to pods.
    1. Select a DA full node. Example: `da-full-2-celestia-node-0`
    1. On that pod, execute `celestia full auth admin` and export it as an env variable locally.
    1. In the sidebar for that pod, scroll down to ports and select forward on the port rpc: 26658/TCP.

3. Generate a proof

    ```shell
    # Change to the correct directory
    cd celestia-zkevm-ibc-demo/provers/blevm/script
    # Run the script
    cargo run
    ```

4. [Optional] To generate a `blevm-mock` proof, modify `script/src/bin/main.rs` with the diff below then run the script again.

    ```diff
    let prover_config = ProverConfig {
    -   elf_bytes: include_elf!("blevm"),
    +   elf_bytes: include_elf!("blevm-mock"),
    };
    ```

## FAQ

How long does it take to generate a proof?

| Proof | Time      | SP1_PROVER |
|-------|-----------|------------|
| blevm | 6 minutes | network    |
