package mpt

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"

	crand "crypto/rand"
	mrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	gethtrie "github.com/ethereum/go-ethereum/trie"
)

type kv struct {
	k, v []byte
	t    bool
}

var prng = initRnd()

func RandomTrie(n int) (trie *gethtrie.Trie, vals map[string]*kv) {
	trie = gethtrie.NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	vals = make(map[string]*kv)
	for i := byte(0); i < 100; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), []byte{i}, false}
		trie.MustUpdate(value.k, value.v)
		trie.MustUpdate(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		trie.MustUpdate(value.k, value.v)
		vals[string(value.k)] = value
	}
	return trie, vals
}

func initRnd() *mrand.Rand {
	var seed [8]byte
	_, err := crand.Read(seed[:])
	if err != nil {
		panic(fmt.Sprintf("failed to read random seed: %v", err))
	}

	rnd := mrand.New(mrand.NewSource(int64(binary.LittleEndian.Uint64(seed[:]))))
	return rnd
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	prng.Read(r)
	return r
}

// Converts proofList to []byte by encoding it with gob
func ProofListToBytes(proof ProofList) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(proof); err != nil {
		return nil, fmt.Errorf("failed to encode proofList to bytes: %w", err)
	}
	return buf.Bytes(), nil
}
