# Celestia Prover

The Celestia Prover is a gRPC service that generates zero-knowledge proofs for Celestia state transitions and data membership. It is designed to work with IBC (Inter-Blockchain Communication) and specifically implements proofs compatible with the ICS-07 Tendermint client specification.

## Usage

> [!WARNING]
> This gRPC service is still under development and may not work as described

To run the server you will need to clone the repo and install rust and cargo.

## Running it in Docker

Before running this program, please follow the steps outlined in this [README.md](https://github.com/celestiaorg/celestia-zkevm-ibc-demo/blob/main/README.md).

After the one-time setup, the following minimum steps are necessary each time you run the program:

1. Spin up the containers including the prover service:

   ```shell
   make start
   ```

   The server will be running (on port `:50051`):

1. Deploy contracts and initialize light clients:

    ```shell
    make setup
    ```

1. Make sure to set `SP1_PROVER=network` in `.env` and get sp1 prover network private key from celestia 1Password.
1. Verify it's running by querying an endpoint.

    ```shell
    grpcurl -plaintext localhost:50051 celestia.prover.v1.Prover/Info
    ```

1. [Optional] Request a proof. Copy the EVM address labeled with `ics07Tendermint` from terminal output which will be used as a `client_id` when querying state transition proofs:

    ```shell
    grpcurl -plaintext -d '{"client_id": ""}' localhost:50051 celestia.prover.v1.Prover/ProveStateTransition
    ```

## Running it locally

When debugging the prover it's much faster to run it locally from the root of the project:

```shell
cargo run -p celestia-prover
```

The setup steps remain the same. Additionally, you need to comment out the prover section in  `docker_compose.yml`

## Protobuf

gRPC depends on proto defined types. These are stored in `proto/prover/v1` from the root directory.

## Contributing

If you update the prover program please make sure that the program works by building the latest Docker image and generating mock proofs.

Build Docker image:

```shell
build-celestia-prover-docker
```

Push new image to GHCR:

```shell
make publish-celestia-prover-docker
```

If you update the circuits please regenerate the `elf` files:

```shell
~/.sp1/bin/cargo-prove prove build --elf-name mock-membership-elf
```
