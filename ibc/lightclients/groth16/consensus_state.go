package groth16

import (
	"time"

	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)

// NewConsensusState returns a new ConsensusState instance.
func NewConsensusState(timestamp time.Time, stateRoot []byte) *ConsensusState {
	return &ConsensusState{
		HeaderTimestamp: timestamppb.New(timestamp),
		StateRoot:       stateRoot,
	}
}

// ClientType returns the groth16 client type.
func (cs *ConsensusState) ClientType() string {
	return Groth16ClientType
}

// GetRoot returns a Merkle root from the current state root.
func (cs *ConsensusState) GetRoot() exported.Root {
	return commitmenttypes.NewMerkleRoot(cs.StateRoot)
}

// GetTimestamp returns the block time (in nanoseconds) of the header that
// created this consensus state.
func (cs *ConsensusState) GetTimestamp() uint64 {
	return uint64(cs.HeaderTimestamp.AsTime().UnixNano())
}

// ValidateBasic is a no-op.
func (cs *ConsensusState) ValidateBasic() error {
	return nil
}

// IsExpired returns true if the provided blockTime is after the unbonding time
// of this consensus state's header timestamp.
func (cs *ConsensusState) IsExpired(blockTime time.Time) bool {
	return cs.HeaderTimestamp.AsTime().Add(unbondingTime).After(blockTime)
}
