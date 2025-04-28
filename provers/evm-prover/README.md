# EVM Prover

The EVM Prover is a gRPC service that generates zero-knowledge proofs for EVM state transitions. It is designed to work with IBC (Inter-Blockchain Communication) and specifically implements proofs compatible with the ICS-07 Tendermint client specification.

## Prerequisites

Before running this program, please follow the steps outlined in this [README.md](https://github.com/celestiaorg/celestia-zkevm-ibc-demo/blob/main/README.md).

## Usage

To run the evm-prover server on port `:50052` from the root directory:

```shell
cargo run --package evm-prover
```

The `Info` endpoint returns the state transition verification key:

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

The proving time for the ZK block range proof depends on the prover settings, i.e. prover network and mode.

For reference in mock mode, the proof generation takes ~15 minutes for aggregating 10 blocks.

## Protobuf

gRPC depends on proto defined types. These are stored in `proto/prover/v1` from the root directory.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for more information.
