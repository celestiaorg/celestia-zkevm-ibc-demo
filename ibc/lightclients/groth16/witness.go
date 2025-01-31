package groth16

import (
	"fmt"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/witness"
)

// PublicWitness should match the public outputs of the SP1 program.
type PublicWitness struct {
	TrustedHeight             int64  // Provided by the relayer/user
	TrustedCelestiaHeaderHash []byte // Provided by the ZK IBC Client
	TrustedRollupStateRoot    []byte // Provided by the ZK IBC Client
	NewHeight                 int64  // Provided by the relayer/user
	NewRollupStateRoot        []byte // Provided by the relayer/user
	NewCelestiaHeaderHash     []byte // Provided by Celestia State Machine
	CodeCommitment            []byte // Provided during initialization of the IBC Client
	GenesisStateRoot          []byte // Provided during initialization of the IBC Client
}

func (p PublicWitness) Generate() (witness.Witness, error) {
	w, err := witness.New(ecc.BN254.ScalarField())
	if err != nil {
		return nil, err
	}

	numInputs := 5

	// Create a channel to send values to the witness
	values := make(chan any, numInputs)
	values <- p
	close(values)

	err = w.Fill(numInputs, 0, values)
	if err != nil {
		return nil, fmt.Errorf("failed to fill witness: %w", err)
	}

	return w, nil
}
