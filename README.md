# ZK-EVM X Celestia Token Transfer Demo

> [!WARNING]
> This repository is a work in progress and under active development.

This repo exists to showcase transferring tokens to and from a Cosmos SDK chain (representing Celestia) and a ZK proveable EVM using the [IBC-Eureka solidity contracts](https://github.com/cosmos/solidity-ibc-eureka/blob/main/README.md). The diagram below is meant to detail the components involved and, at a high level, how they interact with one another.

![mvp-zk-accounts](./mvp-zk-accounts.png)

For more information refer to the [architecture document](./ARCHITECTURE.md). Note that the design is subject to change.

## Local Development

### Prerequisites

1. Install [Docker](https://docs.docker.com/get-docker/)
1. Install [Rust](https://rustup.rs/)
1. Install [Foundry](https://book.getfoundry.sh/getting-started/installation)
1. Install [Bun](https://bun.sh/)
1. Install [Just](https://just.systems/man/en/)
1. Install [SP1](https://docs.succinct.xyz/docs/getting-started/install) (for end-to-end tests)

### Steps

1. Fork this repo and clone it
1. Set up the git submodule for `solidity-ibc-eureka`

    ```shell
    git submodule init
    git submodule update
    ```

1. Install contract dependencies and the SP1 Tendermint light client operator binary from solidity-ibc-eureka.

    ```shell
    make install-dependencies
    ```

1. Start a local development environment

    ```shell
    make start
    ```

1. Set up IBC clients and channels:

    - Generate the `contracts/script/genesis.json` file which contains the initialization parameters for the `SP1ICS07Tendermint` light client contract.
    - Initialize Groth16 light client on simapp.
    - Create a channel on simapp.
    - Deploy IBC contracts on the Reth node.
    - Create a channel on the Reth node.
    - Create a counterparty on the Reth node which points to the groth16 client ID on simapp.
    - Create a counterparty on the simapp which points to the tendermint client ID on Reth.

    ```shell
    make setup
    ```

### Helpful commands

```shell
# See the running containers
docker ps

# You can view the logs from a running container via Docker UI or:
docker logs beacond
docker logs celestia-network-bridge
docker logs celestia-network-validator
docker logs simapp-validator
docker logs reth

# State is persisted in the .tmp directory. Remove .tmp to start fresh:
rm -rf .tmp
```


## Contributing

### Proto Generation

This repo uses protobuf to define the interfaces between several services. To help with this, this
repo relies on [buf](https://buf.build). If you modify the protos you can regenerate them using:

```
buf generate
```