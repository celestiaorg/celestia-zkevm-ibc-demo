# EVM Prover

The EVM Prover is a gRPC service that generates zero-knowledge proofs for EVM state transitions. It is designed to work with IBC (Inter-Blockchain Communication) and specifically implements proofs compatible with the ICS-07 Tendermint client specification.

## Usage

> [!WARNING]
> This gRPC service is still under development and may not lack some features or not work as described.

Before running this program, please follow the steps outlined in this [README.md](https://github.com/celestiaorg/celestia-zkevm-ibc-demo/blob/main/README.md).

To then run the server (on port `:50052`):

    ```shell
    cargo run
    ```

The `Info` endpoint returns the state transition and membership verification keys:

    ```shell
    grpcurl -plaintext localhost:50052 celestia.prover.v1.Prover/Info
    ```

The `ProveStateTransition` endpoint generates a state transition proof for a range of EVM heights:

    ```shell
    grpcurl -plaintext -d '{"client_id":"08-groth16-0"}' localhost:50052 celestia.prover.v1.Prover/ProveStateTransition

    {
    "proof": "...",
    "publicValues": "..."
    }
    ```

Note that this requires the IBC light clients to be setup first:

    ```shell
    make setup
    ```

The proving time for the ZK block range proof depends on the prover settings, i.e. prover network and mode.

For reference in mock mode, the proof generation takes ~15 minutes for aggregating 10 blocks.

## Protobuf

gRPC depends on proto defined types. These are stored in `proto/prover/v1` from the root directory.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for more information.
