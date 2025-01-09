# Celestia Prover

The Celestia Prover is a gRPC service that generates zero-knowledge proofs for Celestia state transitions and data membership. It is designed to work with IBC (Inter-Blockchain Communication) and specifically implements proofs compatible with the ICS-07 Tendermint client specification.

## Usage

> [!WARNING]
> This gRPC service is still under development and may not work as described

To run the server you will need to clone the repo and install rust and cargo.

## Running it in Docker

Before running this program please follow the steps outlined in this [README.md](https://github.com/celestiaorg/celestia-zkevm-ibc-demo/blob/main/README.md)

After the one-time setup each time running the program minimum these steps are necessary:

Spin up the containers including the prover service :

```shell
make start
```

The server will be running (on port `:50051`):

Deploy contracts and initialise light clients:
```shell
make setup
```

Extract the EVM address labeled with ics07Tendermint which will be used as a `client_id` when querying state transition proofs:

```shell
grpcurl -plaintext -d '{"client_id": ""}' localhost:50051 celestia.prover.v1.Prover/ProveStateTransition
```

Groth16 `client_id` is deterministic and after each deployment it'll be - `08-groth16-0`


## Running it locally

The setup steps for this is the same. It also requires commenting out the prover part in `docker_compose.yml`

When debugging the prover it's much faster to run it locally from the root of the project:

```shell
cargo run -p celestia-prover
```

To use the SP1 Prover Network you should also populate the `SP1_PROVER` and `SP1_PRIVATE_KEY` environment variables. You can also use a `.env` file for all environment variables

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
