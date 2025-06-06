# EVM to Celestia Indexer

## Overview

This service indexes the mapping between EVM block heights and Celestia inclusion block heights and blob index in a
[BeaconKit Rollkit rollup](https://github.com/rollkit/beacon-kit/tree/rollkit).

It listens to Celestia blocks, decodes the beacon block in Simple Serialize
(SSZ) format, and provides a queryable API for these mappings. The SSZ
serialized beacon block is stored as the first transaction in the Rollkit
block.

This indexer serves as a temporary stopgap solution. Future versions of
Rollkit will include the data commitment as part of its header, making this external indexing service unnecessary.

## How It Works

* The service connects to a Celestia node via HTTP or WebSocket endpoint
* It monitors for new blocks in the specified namespace
* For each block found, it decodes the Rollkit block data
* It extracts the EVM block number from the first transaction in the Rollkit block
* It stores the mapping (EVM block number → Celestia height) in a local database
* It provides REST API endpoints to query these mappings

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /inclusion_height/{eth_block_number}` | Get the Celestia block height and blob commitment for a specific EVM block number |
| `GET /status` | Get the last processed Celestia block height |
| `GET /health` | Health check endpoint |

## Testing and Verification

To verify the service is running correctly, you can query the health endpoint:

```bash
curl http://localhost:8080/health
```

Expected response:

```bash
OK
```

You can also check the status of the indexer:

```bash
curl http://localhost:8080/status
```

Expected response:

```json
{"last_processed_celestia_height": 864}
```

You can also check if specific EVM blocks have been indexed:

```bash
curl http://localhost:8080/inclusion_height/32
```

Expected response if found:

```json
{"eth_block_number":32,"celestia_height":16,"blob_commitment":"am1pW8KzYUBR2B5KNymVeHxPzw+XfaqkLRSK+Avhq0I="}
```

## Configuration

The service can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `CELESTIA_NODE_URL` | HTTP or WebSocket URL of the Celestia node | `ws://localhost:26658` |
| `CELESTIA_NODE_AUTH_TOKEN` | Authentication token for the Celestia node | `""` (empty string) |
| `CELESTIA_NAMESPACE` | Namespace to monitor for blobs | `0f0f0f0f0f0f0f0f0f0f` |
| `API_PORT` | Port for the HTTP API | `8080` |

## Running the Indexer

### Directly on host

```bash
export CELESTIA_NODE_URL="http://localhost:26658"
export CELESTIA_NODE_AUTH_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
./indexer
go run .
```

### Using Docker

```bash
docker run -p 8080:8080 \
  -e CELESTIA_NODE_URL=ws://your-celestia-node:26658 \
  -e CELESTIA_NODE_AUTH_TOKEN=your-auth-token \
  -v indexer-data:/data \
  eth-celestia-indexer
```
