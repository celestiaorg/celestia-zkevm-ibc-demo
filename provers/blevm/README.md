# blevm

blevm is a service that creates zero-knowledge proofs of EVM state transitions.

## Project layout

This workspace contains multiple crates:

- `blevm`: SP1 program that verifies an EVM block was included in a Celestia data square.
- `blevm-aggregator`: SP1 program that takes as input the verification keys and public values from multiple `blevm` proofs. It verifies the proofs and ensures they are for monotonically increasing EVM blocks.
- `blevm-prover`: library that exposes a `BlockProver` which can generate `blevm` proofs.
- `common`: library with common struct definitions
- `script`: binary that generates a blevm proof for an EVM roll-up block that was posted to Celestia mainnet.

## Contributing

See <https://docs.succinct.xyz/docs/sp1/introduction>

### Prerequisites

1. Install Rust > 1.81.0
1. Create the `.env` file

    ```shell
    cp .env.example .env
    ```

### Usage

The `script` binary will generate an SP1 proof but it depends on a DA node. You can either run a light node locally or proxy to a full node running in Lunaroasis.

1. [Optional] Run a light node locally

    ```shell
    # Initialize a Celestia light node
    celestia light init
    # We need to sync from 4341967 onwards because script/main.rs queries that height.
    # Set the DASer.SampleFrom SampleFrom = 4341967
    vim ~/.celestia-light/config.toml
    # Set the trusted hash to the last block hash.
    # curl -s "https://rpc.celestia.pops.one/block?height=4341967" | jq -r '.result.block.header.last_block_id.hash'
    # 5FA4F4CEF4BA79C1B0854647DB5E331D0746130FCC470FDB7E0E642B4D47EF1E
    export TRUSTED_HASH=5FA4F4CEF4BA79C1B0854647DB5E331D0746130FCC470FDB7E0E642B4D47EF1E
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

    Proofs can be generated using the SP1 prover in either `network` or `mock` mode. `mock` proofs are for testing purposes only. If you'd like to generate real proofs, set the following environment variables:

    ```shell
    SP1_PROVER=network
    # Private key with the permission to use the network prover
    NETWORK_PRIVATE_KEY="" ## the SP1 prover network private key from Celestia 1Password.
    ```

    ```shell
    # Change to the correct directory
    cd celestia-zkevm-ibc-demo/provers/blevm/script

    # Execute blevm
    RUST_LOG=info cargo run --release -- --execute --input-path=input/blevm/1/21991679.bin --inclusion-block=4341967
    # Generate a proof
    RUST_LOG=info cargo run --release -- --prove --input-path=input/blevm/1/21991679.bin --inclusion-block=4341967
    # (Optional) Copy the proof as aggregation input
    cp proof.bin input/blevm-aggregator/1/21991679.bin
    # Generate adjacent header proof
    RUST_LOG=info cargo run --release -- --prove --input-path=input/blevm/1/21991680.bin --inclusion-block=4341969
    # (Optional) Copy the proof as aggregation input
    cp proof.bin input/blevm-aggregator/1/21991680.bin
    ```

4. Aggregate proofs

    ```shell
    # Change to the correct directory
    cd celestia-zkevm-ibc-demo/provers/blevm/script

    # Aggregate proofs
    # Execute blevm aggregator
    RUST_LOG=info cargo run --release --bin blevm-aggregator-script -- --execute --inputs=input/blevm-aggregator/1/21991679.bin --inputs=input/blevm-aggregator/1/21991680.bin
    # Generate aggregation proof
    RUST_LOG=info cargo run --release --bin blevm-aggregator-script -- --prove --inputs=input/blevm-aggregator/1/21991679.bin --inputs=input/blevm-aggregator/1/21991680.bin
    ```

### Other uses

The `blevm-tools` binary can be used to re-create the serialized evm block that is submitted to Celestia by the rollup sequencer.

1. (Optional) Generate client executor input for the block e.g. 18884864 using [rsp](https://github.com/succinctlabs/rsp)

    ```shell
    cd rsp
    RUST_LOG=info cargo run --release --bin rsp -- --block-number 18884864 --chain-id 1 --rpc-url $ETH_RPC_URL --cache-dir=cache
    ```

    The resulting serialized client executor input is included at `script/input/blevm/1/18884864.bin`

2. Dump the serialized evm block bytes from the client executor input into a blob

    ```shell
     cd celestia-zkevm-ibc-demo/provers/blevm/
     RUST_LOG=info cargo run --bin blevm-tools -- --cmd dump-block --input script/input/blevm/1/18884864.bin --output script/blob/blevm/1/18884864.bin
    ```

    The resulting blob is included in [Celestia block number 2988873](https://celenium.io/blob?commitment=eUbPUo7ddF77JSASRuZH1arKP7Ur8PYGtpW0qwvTP0w=&hash=AAAAAAAAAAAAAAAAAAAAAAAAAA8PDw8PDw8PDw8=&height=2988873).

    The inclusion of this blob in Celestia will be verified by the `blevm` sp1 program before verifying the execution of the EVM block. This
    allows us to verify that the correct EVM block was included in the data square and simultaneously verify the correct execution of the EVM block.

### Development

While developing SP1 programs (i.e. `blevm`, `blevm-aggregate`) it is helpful to generate [development builds](https://docs.succinct.xyz/docs/sp1/writing-programs/compiling#development-builds):

```shell
# Change to an SP1 program crate
cd blevm
# Build for development
cargo prove build
```

## FAQ

How long does it take to generate a proof?

| SP1_PROVER | Program    | Time       |
|------------|------------|------------|
| network    | blevm      | 6 minutes  |
| mock       | blevm      | 90 seconds |
