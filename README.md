# ZK-EVM X Celestia Token Transfer Demo

> [!WARNING]
> This repository is a work in progress and under active development.

This repo exists to showcase transferring tokens to and from a Cosmos SDK chain (representing Celestia) and a ZK proveable EVM using the [IBC-Eureka solidity contracts](https://github.com/cosmos/solidity-ibc-eureka/blob/main/README.md). The diagram below is meant to detail the components involved and, at a high level, how they interact with one another.

![mvp-zk-accounts](./docs/images/mvp-zk-accounts.png)

For more information refer to the [architecture document](./ARCHITECTURE.md). Note that the design is subject to change.

## Usage

### Prerequisites

1. Install [Docker](https://docs.docker.com/get-docker/)
1. Install [Rust](https://rustup.rs/)
1. Install [Foundry](https://book.getfoundry.sh/getting-started/installation)
1. Install [Bun](https://bun.sh/)
1. Install [Just](https://just.systems/man/en/)
1. Install [SP1](https://docs.succinct.xyz/docs/getting-started/install)

### Steps

1. Fork this repo and clone it
1. Set up the git submodule for `solidity-ibc-eureka`

    ```shell
    git submodule init
    git submodule update
    ```

1. Apply this diff to the `solidity-ibc-eureka/justfile`:

    ```diff
    genesis-sp1-ics07: build-sp1-programs
    @echo "Generating the genesis file..."
    -  RUST_LOG=info cargo run --bin operator --release -- genesis -o scripts/genesis.json
    +  RUST_LOG=info cargo run --bin operator --release -- genesis --proof-type groth16 -o scripts/genesis.json
    +  @echo "--> Setting the verifier key in scripts/genesis.json"
    +  @sed -i '' 's|"updateClientVkey": "0x00d003b09381282af2781e5ec015aae610d766a08fffd4ac45d2e6dad736ead3"|"updateClientVkey": "0x0081282e1279b92586a3330c1cd83a3f0910299bdc7be90ac1a24b2a0826523b"|' scripts/genesis.json
    +  @echo "--> Set the verifier key to 0x0081282e1279b92586a3330c1cd83a3f0910299bdc7be90ac1a24b2a0826523b."
    ```

1. Create and populate the `.env` file

    ```shell
    cp .env.example .env
    # Modify the .env file and set `SP1_PROVER=network` and `NETWORK_PRIVATE_KEY="PRIVATE_KEY"` to the SP1 prover network private key from Celestia 1Password.
    ```

1. Modify the `docker-compose.yml` file and set `SP1_PROVER=network` and `NETWORK_PRIVATE_KEY="PRIVATE_KEY"` to the SP1 prover network private key from Celestia 1Password.

    ```diff
    celestia-prover:
        image: ghcr.io/celestiaorg/celestia-zkevm-ibc-demo/celestia-prover:latest
        container_name: celestia-prover
        environment:
        # TENDERMINT_RPC_URL should be the SimApp which is acting as a substitute
        # for Celestia (with IBC Eurekea enabled).
        - TENDERMINT_RPC_URL=http://simapp-validator:26657
        - RPC_URL=http://reth:8545
        - PROTO_DESCRIPTOR_PATH=proto_descriptor.bin
    +      - SP1_PROVER=network
    +      - SP1_PRIVATE_KEY=PRIVATE_KEY
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
    - Initialize Groth16 light client on SimApp.
    - Create a channel on SimApp.
    - Deploy IBC contracts on the Reth node.
    - Create a channel on the Reth node.
    - Create a counterparty on the Reth node which points to the groth16 client ID on SimApp.
    - Create a counterparty on the SimApp which points to the tendermint client ID on Reth.

    ```shell
    make setup
    ```

1. Run the demo

    ```shell
    # Transfer tokens from SimApp to the EVM roll-up.
    make transfer
    # Relay the token transfer
    make relay
    # To stop and teardown the test environment (when you're finished)
    make stop
    ```

## A Breakdown of an IBC Transfer

This section takes the diagram from above and breaks down each step during `make transfer` to help aid your understanding.

![mvp-zk-accounts](./docs/images/mvp-zk-accounts-step-1.png)

1a -> The user submits a transfer message. This is a `ICS20LibFungibleTokenPacketData` wrapped in a `SendPacket` message. As well as who it's sending the tokens to and how much it also specifies where this packet is going to and lets the eventual receiver know where the packet came from.

1b -> The SimApp chain (mimicking Celestia) executes the transaction, checking the user's balance and then moving the funds to a locked acount. It stores a commitment to this execution in state. This is kind of like a verifiable receipt.

![mvp-zk-accounts](./docs/images/mvp-zk-accounts-step-2.png)

2a -> Now the relayer kicks in. It listens to events that SimApp has emitted that there are pending packets ready to be sent to other chains. It queries the chain for the receipt based on a predetermined location.

2b -> The relayer needs to prove to the EVM rollup that SimApp has actually successfully executed the first part of the transfer: locking up the tokens. Proving this requires two steps: First the relayer queries a state transition proof from the prover process. This will prove the latest state root from the last trusted state root stored in the state of the ICS07 Tendermint smart contract on the EVM. Now the EVM has an up to date record of SimApp's current state (which includes the receipt). Second, the relayer asks the prover for a proof that the receipt is a merkle leaf of the state root i.e. it's part of state

2c -> The prover has a zk circuit for generating both proofs. One takes tendermint headers and uses the `SkippingVerification` algorithm to assert the latest header. The other takes IAVL merkle proofs and proves some leaf key as part of the root. These are both STARK proofs which can be processed by the smart contracts on the EVM.

2d -> The last step of the relayer is to combine these proofs and packets and submit a `MsgUpdateClient` and `MsgRecvPacket` to the EVM rollup.

![mvp-zk-accounts](./docs/images/mvp-zk-accounts-step-3.png)

Step 3 mirrors step 2 in many ways but now in the opposite direction

3a -> The EVM executes both messages. It verifies the STARK proofs and updates it's local record of SimApp's state. It then uses the updated state to verify that the receipt that the packet refers is indeed present in SimApp's state. Once all the verification checks are passed. It mints the tokens and adds them to the account of the recipient as specified in the packet. The rollup then writes it's own respective receipt that it processed the corresponding message.

3b -> Similarly, the relayer listens for events emitted from the EVM rollup for any packets awaiting to be sent back. Upon receiving the packet to be returned, an acknowledgement of the transfer to be sent back to SimApp, it talks to the prover service to prepare the relevant proofs. While they are of different state machines and different state trees, the requests are universal: a proof of the state transition and a proof of membership. The EVM Prover Service here futher compresses the STARK proofs into groth16 proofs for SimApp's groth16 IBC Client.

3c -> The relayer then sends a `MsgUpdateClient` with the state transition proof to update SimApp's record of the Rollup's state after the point that it processed the transfer packet and wrote the receipt. The relayer also sends a `MsgAcknowledgement` which contains the membership proof of the commitment, a.k.a. the receipt alongside the details of the receipt i.e. for what transfer message are we acknowledging.

3d -> SimApp processes these two messages. It validates the proofs and if everything is in order, it removes the transfer receipt and keeps one final receipt of the acknowledgement (to prevent a later timeout message).

In the case that the EVM decided these messages were not valid it would not write the acknowledgement receipt. The relayer, tracking the time when the transfer message was sent would submit a `MsgTimeout` instead of the acknowledgement with an absence proof. This is a proof that no acknowledgement was written where the predermined path says it should be written. When SimApp receives this timeout and the corresponding absence proof, it reverses the transfer, releaseing the locked funds and returning them to the sender. This process is atomic - funds can not be unlocked if they are minted on the other chain.

If someone were to send tokens from the EVM rollup back to SimApp, the source chain of those tokens, the process would be very similar, however the actions wouldn't be to lock and mint but rather the EVM rollup would burn tokens and SimApp would unlock them.

## Contributing

### Proto Generation

This repo uses protobuf to define the interfaces between several services. To help with this, this
repo relies on [buf](https://buf.build). If you modify the protos you can regenerate them using:

```shell
make proto-gen
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
```
