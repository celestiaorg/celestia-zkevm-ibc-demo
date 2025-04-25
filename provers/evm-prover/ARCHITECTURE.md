# EVM Prover

The EVM Prover implements the Prover service for EVM rollups, creating zero-knowledge proofs for state membership and state transitions. It uses SP1 programs (`blevm` and `blevm-aggregator`) to generate ZK proofs of data inclusion in Celestia and EVM execution using the Rollup State Prover (RSP).

## IBC Transfer Architecture

The EVM Prover is a key component in the IBC transfer architecture, enabling secure asset transfers between Celestia and EVM rollups.

```mermaid
sequenceDiagram
    participant User
    participant SimApp
    participant Relayer
    participant CelestiaProver
    participant EVMProver
    participant EVM
    
    %% Initial Transfer: SimApp to EVM
    User->>SimApp: Submit MsgTransfer (ICS20)
    SimApp->>SimApp: Execute transaction
    Note over SimApp: Check balance & lock funds
    SimApp->>SimApp: Store commitment in state
    
    %% State Transition Proof for EVM
    SimApp-->>Relayer: Emit SendPacket events
    Relayer->>CelestiaProver: Query state transition proof
    CelestiaProver-->>Relayer: Return SP1 state transition proof
    Relayer->>EVM: Submit MsgUpdateClient with state transition proof
    EVM->>EVM: Verify SP1 proof
    EVM->>EVM: Update Tendermint light client
    
    %% State Membership Proof for EVM
    Relayer->>CelestiaProver: Query membership proof
    CelestiaProver-->>Relayer: Return SP1 membership proof
    Relayer->>EVM: Submit MsgRecvPacket with membership proof
    EVM->>EVM: Verify receipt in SimApp state
    EVM->>EVM: Mint tokens to recipient
    EVM->>EVM: Store receipt
    
    %% Acknowledgement: EVM to SimApp
    EVM-->>Relayer: Emit WriteAcknowledgement events
    Relayer->>EVMProver: Query state transition proof
    EVMProver-->>Relayer: Return Groth16 state transition proof
    Relayer->>SimApp: Submit MsgUpdateClient with state transition proof
    SimApp->>SimApp: Verify Groth16 proof
    SimApp->>SimApp: Update EVM light client
    
    %% State Membership Proof for SimApp
    Relayer->>EVMProver: Query membership proof
    EVMProver-->>Relayer: Return Groth16 membership proof
    Relayer->>SimApp: Submit MsgAcknowledgement with membership proof
    SimApp->>SimApp: Verify receipt in EVM state
    SimApp->>SimApp: Unlock or burn tokens based on ack
    SimApp->>SimApp: Store acknowledgement
```

## Architecture

```mermaid
graph TD
    subgraph "EVM Rollup"
        RollupNode["Rollup Node"]
        IBC["IBC Eureka Smart Contracts"]
        Sequencer["Centralized Sequencer"]
    end
    
    subgraph "Prover System"
        EVMProver["EVM Prover Service"]
        SP1Programs["SP1 Programs"]
        SP1Programs --> BLEVM["blevm (Block Prover)"]
        SP1Programs --> BLEVMAgg["blevm-aggregator"]
    end
    
    subgraph "Indexing"
        RollkitIndexer["Rollkit Indexer"]
        BlockStorage["Block Data Storage"]
    end
    
    subgraph "Data Sources"
        CelestiaDA["Celestia DA"]
        EVMState["EVM State"]
    end
    
    subgraph "IBC Infrastructure"
        Relayer["IBC Relayer"]
        CelestiaProver["Celestia Prover"]
    end
    
    %% Data flow connections
    CelestiaDA -->|"Raw blocks"| RollkitIndexer
    RollupNode -->|"Block execution data"| EVMState
    RollkitIndexer -->|"Height to inclusion mapping"| BlockStorage
    
    EVMState -->|"State data"| EVMProver
    BlockStorage -->|"Raw block data"| EVMProver
    
    EVMProver -->|"Uses"| BLEVM
    EVMProver -->|"Uses"| BLEVMAgg
    
    Relayer -->|"ProveStateTransition request"| EVMProver
    Relayer -->|"ProveStateMembership request"| EVMProver
    EVMProver -->|"Groth16 proofs"| Relayer
    
    Relayer -->|"Submit proofs + packets"| IBC
    Sequencer -->|"Transactions"| RollupNode
```

## Flow Overview

The EVM Prover implements the Prover service protocol to provide zero-knowledge proofs for:
1. State membership verification
2. State transition verification

These proofs are used within the IBC (Inter-Blockchain Communication) protocol to securely transfer assets between the EVM rollup and other chains.

```mermaid
graph TD
    subgraph "State Transition Proof Flow"
        RelayerST[Relayer]
        EVMProverST[EVM Prover]
        IndexerST[Rollkit Indexer]
        BLEVMST[blevm SP1 Program]
        BLEVMAggST[blevm-aggregator]
        
        RelayerST -->|"1. ProveStateTransition request"| EVMProverST
        EVMProverST -->|"2. Query blocks in range"| IndexerST
        IndexerST -->|"3. Return block data"| EVMProverST
        EVMProverST -->|"4. Generate proofs for each block"| BLEVMST
        BLEVMST -->|"5. Return BlevmOutputs"| EVMProverST
        EVMProverST -->|"6. Aggregate proofs"| BLEVMAggST
        BLEVMAggST -->|"7. Return BlevmAggOutput"| EVMProverST
        EVMProverST -->|"8. Return proof response"| RelayerST
    end
    
    subgraph "State Membership Proof Flow"
        RelayerSM[Relayer]
        EVMProverSM[EVM Prover]
        IndexerSM[Rollkit Indexer]
        BLEVMSM[blevm SP1 Program]
        
        RelayerSM -->|"1. ProveStateMembership request"| EVMProverSM
        EVMProverSM -->|"2. Query block at height"| IndexerSM
        IndexerSM -->|"3. Return block data"| EVMProverSM
        EVMProverSM -->|"4. Generate proof with key path"| BLEVMSM
        BLEVMSM -->|"5. Return BlevmOutput"| EVMProverSM
        EVMProverSM -->|"6. Return proof response"| RelayerSM
    end
```

## Prover Service

The EVM Prover implements the following gRPC service:

```protobuf
service Prover {
  rpc Info(InfoRequest) returns (InfoResponse);
  rpc ProveStateTransition(ProveStateTransitionRequest) returns (ProveStateTransitionResponse);
  rpc ProveStateMembership(ProveStateMembershipRequest) returns (ProveStateMembershipResponse);
}
```

### Endpoints

#### Info
Returns information about the prover, including verifier keys.

#### ProveStateTransition
Generates a zero-knowledge proof of state transition for a given client ID (Tendermint light client contract address for EVM chains).

#### ProveStateMembership
Generates a zero-knowledge proof of state membership for a given client ID and key paths.

## SP1 Programs

### blevm

The `blevm` SP1 program generates proofs of data inclusion and execution for single EVM blocks.

#### Inputs:
- `KeccakInclusionToDataRootProofInput`: Data for inclusion proof
- `EthClientExecutorInput`: Data for EVM execution
- `celestia_header_hash`: Celestia block header hash
- `data_hash_bytes`: Hash of the block data
- `proof_data_hash_to_celestia_hash`: Merkle proof connecting data hash to Celestia hash

#### Output:
```rust
pub struct BlevmOutput {
    pub blob_commitment: [u8; 32],
    pub header_hash: [u8; 32],
    pub prev_header_hash: [u8; 32],
    pub height: u64,
    pub gas_used: u64,
    pub beneficiary: [u8; 20],
    pub state_root: [u8; 32],
    pub celestia_header_hash: [u8; 32],
}
```

### blevm-aggregator

The `blevm-aggregator` SP1 program aggregates multiple block proofs to create a proof over a range of blocks.

#### Inputs:
- `vkeys`: Verification keys for the proofs to aggregate
- `public_values`: Public values from the proofs to aggregate
- Proofs (automatically input)

#### Output:
```rust
pub struct BlevmAggOutput {
    pub newest_header_hash: [u8; 32],
    pub oldest_header_hash: [u8; 32],
    pub celestia_header_hashes: Vec<[u8; 32]>,
    pub newest_state_root: [u8; 32],
    pub newest_height: u64,
}
```

## Rollkit Indexer

The current architecture uses an indexer for Rollkit which:
- Maps EVM block heights to their inclusion heights in Celestia
- Stores raw block data submitted to Celestia
- Provides these as inputs for proving inclusion

This mapping is crucial for the EVM Prover to locate the correct Celestia blocks when generating proofs.

## Current Limitations and Future Work

### Current Limitations

1. **Matching Inclusion and Execution Data**: The system for matching inclusion data with execution data is not yet implemented.
2. **Sequencer Signature Verification**: Verification of sequencer signatures is pending implementation.

### Future Improvements

1. **Modular Proofs**: In the future, we should implement modular proofs that can:
   - Create inclusion, execution, and signature verification proofs in parallel
   - Verify proofs over a requested range in parallel
   - Improve throughput and reduce latency for proof generation

2. **Enhanced Indexer**: Improve the indexer to provide more efficient lookups and support for additional metadata.

3. **Optimized Proof Aggregation**: Implement more efficient algorithms for proof aggregation to reduce computational overhead.

## Input Data for Block Proving

```rust
pub struct BlockProverInput {
    pub inclusion_height: u64,
    pub client_executor_input: Vec<u8>,
    pub rollup_block: Vec<u8>,
}
```

## Aggregation Interface

### Input:
```rust
pub struct AggregationInput {
    pub proof: SP1ProofWithPublicValues,
    pub vk: SP1VerifyingKey,
}
```

### Output:
```rust
pub struct AggregationOutput {
    pub proof: SP1ProofWithPublicValues,
}
```

## Integration with IBC

The EVM Prover works in conjunction with the IBC Eureka Smart Contracts and Relayer to facilitate cross-chain communication and asset transfers. The proofs generated by the EVM Prover are used to verify state transitions and state membership across chains, ensuring security and integrity in cross-chain operations.
