package groth16

import (
	"fmt"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/witness"
)

// PublicWitness should match the public outputs of the SP1 program.
type PublicWitness struct {
	// NewestHeaderHash is the last block's hash on the EVM roll-up
	NewestHeaderHash []byte
	// OldestHeaderHash is the earliest block's hash on the EVM roll-up
	OldestHeaderHash []byte
	// CelestiaHeaderHashes is the range of Celestia blocks that include all
	// of the blob data the EVM roll-up has posted from oldest_header_hash to
	// newest_header_hash
	CelestiaHeaderHashes [][]byte
	// NewestStateRoot is the computed state root of the EVM roll-up after
	// processing blocks from oldest_header_hash to newest_header_hash
	NewestStateRoot []byte
	// NewestHeight is the most recent block number of the EVM roll-up
	NewestHeight uint64
}

func (p PublicWitness) Generate() (witness.Witness, error) {
	w, err := witness.New(ecc.BN254.ScalarField())
	if err != nil {
		return nil, err
	}

	// Convert each field to a field element
	values := make(chan any, 5)

	// Convert NewestHeaderHash to field element
	values <- p.NewestHeaderHash

	// Convert OldestHeaderHash to field element
	values <- p.OldestHeaderHash

	// Convert CelestiaHeaderHashes to field elements
	for _, hash := range p.CelestiaHeaderHashes {
		values <- hash
	}

	// Convert NewestStateRoot to field element
	values <- p.NewestStateRoot

	// Convert NewestHeight to field element
	values <- p.NewestHeight

	close(values)

	err = w.Fill(5, 0, values)
	if err != nil {
		return nil, fmt.Errorf("failed to fill witness: %w", err)
	}

	return w, nil
}
