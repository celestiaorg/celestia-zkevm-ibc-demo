package groth16

import (
	"context"
	"errors"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// VerifyUpgradeAndUpdateState returns an error because it hasn't been
// implemented yet.
func (cs *ClientState) VerifyUpgradeAndUpdateState(
	// QUESTION: should we change this to context.Context?
	ctx context.Context,
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	upgradedClient exported.ClientState,
	upgradedConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
	return errors.New("not implemented")
}
