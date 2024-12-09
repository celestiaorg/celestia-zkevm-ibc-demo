# ZK-EVM X Celestia Token Transfer Demo

> ⚠️ **Warning**
> This repository is a work in progress and under active development.

This repo exists to showcase transferring tokens to and from a Cosmos SDK chain (representing Celestia) and a ZK proveable EVM using the [IBC-Eureka solidity contracts](https://github.com/cosmos/solidity-ibc-eureka/tree/main/src). The diagram below is meant to detail the components involved and, at a high level, how they interact with one another.

![mvp-zk-accounts](./mvp-zk-accounts.png)

For more information refer to the [architecture document](./ARCHITECTURE.md). Note that the design is subject to change.

## Contributing

1. Complete the solidity-ibc-eureka [requirements](https://github.com/cosmos/solidity-ibc-eureka?tab=readme-ov-file#requirements)
    1. After you `cp .env.example .env` you will need to set the `PRIVATE_KEY`. You can create a new account in Metamask and export the private key to use.
1. Install [Docker](https://docs.docker.com/get-docker/)
1. Fork this repo and clone it
1. Set up the git submodule for solidity-ibc-eureka via:

    ```shell
    git submodule init
    git submodule update
    ```

### Local development

```shell
# Start a local development environment by running
docker compose up --detach

# Deploy smart contracts on the EVM roll-up
make deploy-contracts

# Build the demo binary
make build-demo

# Run the demo binary
# NOTE: this currently doesn't work
make run-demo
```

While deploying contracts, if you hit an error like: `[Revert] vm.envString: environment variable "E2E_FAUCET_ADDRESS" not found` then comment out the lines that use that environment variable from `./solidity-ibc-eureka/E2ETestDeploy.s.sol`.

### Helpful commands

```shell
# See the running contains
docker ps

# You can view the logs from a running container via Docker UI or:
docker logs beacond
docker logs celestia-network-bridge
docker logs celestia-network-validator
docker logs simapp-validator
docker logs reth
```
