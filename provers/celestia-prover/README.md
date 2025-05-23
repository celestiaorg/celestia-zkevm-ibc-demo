# Celestia Prover

The Celestia Prover is a gRPC service that generates zero-knowledge proofs for Celestia state transitions and data membership. It is designed to work with IBC (Inter-Blockchain Communication) and specifically implements proofs compatible with the ICS-07 Tendermint client specification.

## Usage

To run the server you will need to clone the repo and install rust and cargo.

## Running it in Docker

Before running this program, please follow the steps outlined in this [README.md](https://github.com/celestiaorg/celestia-zkevm-ibc-demo/blob/main/README.md).

1. Build the docker image:

    ```shell
    build-celestia-prover-docker
    ```

1. Run the docker containers:

    ```shell
    make start
    ```

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

The setup steps remain the same. Additionally, you need to comment out the prover section in `docker_compose.yml`

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

## Benchmarks

How long does it take to generate a proof?

| Proof            | Time       | Cycles |
|------------------|------------|--------|
| State Transition | 80s - 235s | TBD    |
| Membership       | 105s       | TBD    |
