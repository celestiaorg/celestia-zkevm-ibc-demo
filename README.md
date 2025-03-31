# ZK EVM IBC demo

> [!WARNING]
> This repository is a work in progress and under active development.

This repo exists to showcase transferring tokens between SimApp (a Cosmos SDK chain representing Celestia) and a ZK proveable EVM via [IBC V2](https://github.com/cosmos/ibc/blob/main/spec/IBC_V2/README.md) (formerly known as IBC Eureka) and the IBC V2 [solidity contracts](https://github.com/cosmos/solidity-ibc-eureka/blob/main/README.md). The diagram below is meant to detail the components involved and, at a high level, how they interact with one another.

![mvp-zk-accounts](./docs/images/mvp-zk-accounts.png)

For more information refer to the [architecture](./docs/ARCHITECTURE.md). Note that the design is subject to change.

## Usage

### Prerequisites

1. Install [Docker](https://docs.docker.com/get-docker/)
1. Install [Rust](https://rustup.rs/)
1. Install [Foundry](https://book.getfoundry.sh/getting-started/installation)
1. Install [Bun](https://bun.sh/)
1. Install [Just](https://just.systems/man/en/)
1. Install [SP1](https://docs.succinct.xyz/docs/sp1/getting-started/install)

### Steps

1. Fork this repo and clone it
1. Set up the git submodule for `solidity-ibc-eureka`

    ```shell
    git submodule init
    git submodule update
    ```

1. Create and populate the `.env` file in this repo

    ```shell
    cp .env.example .env
    # Modify the .env file:
    # Set SP1_PROVER=network
    # Set NETWORK_PRIVATE_KEY="PRIVATE_KEY" to the SP1 prover network private key from Celestia 1Password
    ```

1. Create and populate the `.env` file in solidity-ibc-eureka

    ```shell
    cd solidity-ibc-eureka
    cp .env.example .env
    # Modify the .env file:
    # Set VERIFIER=""
    ```

1. Modify the `docker-compose.yml` file and set `NETWORK_PRIVATE_KEY="PRIVATE_KEY"` to the SP1 prover network private key from Celestia 1Password.

    ```diff
    celestia-prover:
        image: ghcr.io/celestiaorg/celestia-zkevm-ibc-demo/celestia-prover:latest
        container_name: celestia-prover
        environment:
        # TENDERMINT_RPC_URL should be the SimApp which is acting as a substitute
        # for Celestia (with IBC Eurekea enabled).
        - TENDERMINT_RPC_URL=http://simapp-validator:26657
        - RPC_URL=http://reth:8545
        - CELESTIA_PROTO_DESCRIPTOR_PATH=proto_descriptor.bin
        - SP1_PROVER=network
    +   - NETWORK_PRIVATE_KEY=PRIVATE_KEY
    ```

1. Install contract dependencies and the SP1 Tendermint light client operator binary from solidity-ibc-eureka.

    ```shell
    make install-dependencies
    ```

1. Run the demo

    ```shell
    # This runs make start, setup, and transfer
    make demo
    ```

## Architecture

See [ARCHITECTURE.md](./docs/ARCHITECTURE.md) for more information.

## Contributing

See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for more information.
