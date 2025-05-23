package mpt

import (
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// VerifyMerklePatriciaTrieProof verifies MPT proofs with IBC public inputs
func VerifyMerklePatriciaTrieProof(rootHash ethcommon.Hash, key []byte, proof []hexutil.Bytes) (value []byte, err error) {
	proofDB, err := ReconstructProofDB(proof)
	if err != nil {
		return nil, fmt.Errorf("failed to decode proof: %w", err)
	}
	return trie.VerifyProof(rootHash, key, proofDB)
}

// ReconstructProofDB iterates over all nodes in proof and returns a database of key value pairs where:
// - key is the node hash if the node is larger than 32 bytes, otherwise the node itself
// - value is the node encoded as hex
func ReconstructProofDB(proof []hexutil.Bytes) (ethdb.Database, error) {
	proofDB := rawdb.NewMemoryDatabase()
	for i, encodedNode := range proof {
		nodeKey := encodedNode
		if len(encodedNode) >= 32 { // small MPT nodes are not hashed
			nodeKey = crypto.Keccak256(encodedNode)
		}
		if err := proofDB.Put(nodeKey, encodedNode); err != nil {
			return nil, fmt.Errorf("failed to load proof node %d into mem db: %w", i, err)
		}
	}

	return proofDB, nil
}
