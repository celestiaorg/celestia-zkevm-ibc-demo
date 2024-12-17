# ZK-EVM X Celestia Token Transfer Demo

> ⚠️ **Warning**
> This repository is a work in progress and under active development.

This repo exists to showcase transferring tokens to and from a Cosmos SDK chain (representing Celestia) and a ZK proveable EVM using the [IBC-Eureka solidity contracts](https://github.com/cosmos/solidity-ibc-eureka/tree/main/src). The diagram below is meant to detail the components involved and, at a high level, how they interact with one another.

![mvp-zk-accounts](./mvp-zk-accounts.png)

For more information refer to the [architecture document](./ARCHITECTURE.md). Note that the design is subject to change.

## Local Development

### Prerequisites

1. Install [Docker](https://docs.docker.com/get-docker/)
1. Install [Rust](https://rustup.rs/)
1. Install [Foundry](https://book.getfoundry.sh/getting-started/installation)
1. Install [Bun](https://bun.sh/)
1. Install [Just](https://just.systems/man/en/)
1. Install [SP1](https://succinctlabs.github.io/sp1/getting-started/install.html) (for end-to-end tests)

### Steps

1. Fork this repo and clone it
1. Set up the git submodule for `solidity-ibc-eureka`

    ```shell
    git submodule init
    git submodule update
    ```

1. Start a local development environment

    ```shell
    docker compose up --detach
    ```

    > [!TIP]: Double check that all 5 containers are started. Currently: the bridge might fail because it depends on the validator being available. If this happens, wait until the validator is available then start the bridge and beacond node again.

1. Copy the `.env` file into `./solidity-ibc-eureka`

    ```shell
    cp .env.example .solidity-ibc-eureka/.env
    ```

1. Deploy the Tendermint light client smart contract on the EVM roll-up. Note: this may not be necessary.

    ```shell
    cd solidity-ibc-eureka && just deploy-sp1-ics07
    ```

1. Deploy smart contracts on the EVM roll-up.

    ```shell
    make deploy-contracts
    ```

    > [!TIP]: While deploying contracts, if you hit an error like: `[Revert] vm.envString: environment variable "E2E_FAUCET_ADDRESS" not found` then comment out the lines that use that environment variable from `./solidity-ibc-eureka/E2ETestDeploy.s.sol`.

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