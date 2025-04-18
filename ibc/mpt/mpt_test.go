package mpt

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	gethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestVerifyMerklePatriciaTrieProof(t *testing.T) {
	trie, vals := RandomTrie(4)
	root := trie.Hash()
	for i, prover := range makeProvers(trie) {

		for _, kv := range vals {
			proof, err := prover(kv.k)
			require.NoError(t, err)
			require.NotNil(t, proof)
			fmt.Printf("proof list proof example: %v\n", proof)
			proofBytes, err := ProofListToBytes(*proof)
			fmt.Printf("proof bytes: %x\n", proofBytes)
			require.NoError(t, err)
			val, err := VerifyMerklePatriciaTrieProof(root, kv.k, proofBytes)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x", i, kv.k, err, proof)
			}
			if !bytes.Equal(val, kv.v) {
				t.Fatalf("prover %d: verified value mismatch for key %x: have %x, want %x", i, kv.k, val, kv.v)
			}
		}
	}
}

// makeProvers creates Merkle trie provers based on different implementations to
// test all variations.
func makeProvers(trie *gethtrie.Trie) []func(key []byte) (*ProofList, error) {
	var provers []func(key []byte) (*ProofList, error)

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) (*ProofList, error) {
		var proof ProofList
		err := trie.Prove(key, &proof)
		if err != nil {
			return nil, err
		}
		return &proof, nil
	})
	// Create a leaf iterator based Merkle prover
	provers = append(provers, func(key []byte) (*ProofList, error) {
		var proof ProofList
		if it := gethtrie.NewIterator(trie.MustNodeIterator(key)); it.Next() && bytes.Equal(key, it.Key) {
			for _, p := range it.Prove() {
				err := proof.Put(crypto.Keccak256(p), p)
				if err != nil {
					return nil, err
				}
			}
		}
		return &proof, nil
	})
	return provers
}
