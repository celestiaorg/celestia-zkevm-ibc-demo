package groth16

// BlevmAggOutput is the output of the Blevm proof aggregator which gets used as a public
// witness for proof verification. 
// Ref: https://github.com/celestiaorg/celestia-zkevm-ibc-demo/blob/4c43989012340c400d525751c32ef1a2d7762e8f/provers/blevm/common/src/lib.rs#L16
type BlevmAggOutput struct {
	// newest_header_hash is the last block's hash on the EVM roll-up
	NewestHeaderHash [32]byte
	// oldest_header_hash is the earliest block's hash on the EVM roll-up
	OldestHeaderHash [32]byte
	// celestia_header_hashes is the range of Celestia blocks that include all
	// of the blob data the EVM roll-up has posted from oldest_header_hash to
	// newest_header_hash
	CelestiaHeaderHashes [][]byte
	// newest_state_root is the computed state root of the EVM roll-up after
	// processing blocks from oldest_header_hash to newest_header_hash
	NewestStateRoot [32]byte
	// newest_height is the most recent block number of the EVM roll-up
	NewestHeight uint64
}
