package groth16

import (
	"bytes"
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VerifyClientMessage checks if the clientMessage is of type Header
func (cs *ClientState) VerifyClientMessage(ctx context.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, clientStore, cdc, msg)
	default:
		return clienttypes.ErrInvalidClientType
	}
}

func (cs *ClientState) verifyHeader(_ context.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, header *Header) error {
	// sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/5917

	// get consensus state from clientStore for trusted height
	_, err := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, uint64(header.TrustedHeight)))
	if err != nil {
		return sdkerrors.Wrapf(
			err, "could not get consensus state from clientstore at TrustedHeight: %d", header.TrustedHeight,
		)
	}

	// assert header height is newer than consensus state
	if header.GetHeight().LTE(clienttypes.NewHeight(0, uint64(header.TrustedHeight))) {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader,
			"header height ≤ consensus state height (%d ≤ %d)", header.GetHeight(), header.TrustedHeight,
		)
	}

	return nil
}

// * (modules/core/02-client) [\#1210](https://github.com/cosmos/ibc-go/pull/1210)
// Removing `CheckHeaderAndUpdateState` from `ClientState` interface & associated light client implementations.

// CheckHeaderAndUpdateState checks if the provided header is valid, and if valid it will:
// create the consensus state for the header.Height
// and update the client state if the header height is greater than the latest client state height
// It returns an error if:
// - the client or header provided are not parseable to tendermint types
// - the header is invalid
// - header height is less than or equal to the trusted header height
// - header revision is not equal to trusted header revision
// - header valset commit verification fails
// - header timestamp is past the trusting period in relation to the consensus state
// - header timestamp is less than or equal to the consensus state timestamp
//
// UpdateClient may be used to either create a consensus state for:
// - a future height greater than the latest client state height
// - a past height that was skipped during bisection
// If we are updating to a past height, a consensus state is created for that height to be persisted in client store
// If we are updating to a future height, the consensus state is created and the client state is updated to reflect
// the new latest height
// UpdateClient must only be used to update within a single revision, thus header revision number and trusted height's revision
// number must be the same. To update to a new revision, use a separate upgrade path
// Tendermint client validity checking uses the bisection algorithm described
// in the [Tendermint spec](https://github.com/tendermint/spec/blob/master/spec/consensus/light-client.md).
//
// Pruning:
// UpdateClient will additionally retrieve the earliest consensus state for this clientID and check if it is expired. If it is,
// that consensus state will be pruned from store along with all associated metadata. This will prevent the client store from
// becoming bloated with expired consensus states that can no longer be used for updates and packet verification.
func (cs *ClientState) UpdateState(
	ctx context.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		// clientMsg is invalid Misbehaviour, no update necessary
		return []exported.Height{}
	}

	// performance: do not prune in checkTx
	// simulation must prune for accurate gas estimation
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/5917

	// check for duplicate update
	// TODO: remove this because header is validated in VerifyClientMessage
	consensusState, err := GetConsensusState(clientStore, cdc, header.GetHeight())
	if err != nil {
		panic(fmt.Sprintf("failed to retrieve consensus state: %s", err))
	}
	if consensusState != nil {
		// perform no-op
		return []exported.Height{header.GetHeight()}
	}

	trustedConsensusState, err := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, uint64(header.TrustedHeight)))
	if err != nil {
		panic(fmt.Sprintf("failed to retrieve trusted consensus state: %s", err))
	}

	vk, err := DeserializeVerifyingKey(cs.StateTransitionVerifierKey)
	if err != nil {
		return []exported.Height{}
	}

	// initialize the public witness
	publicWitness := PublicWitness{
		TrustedHeight:             header.TrustedHeight,
		TrustedCelestiaHeaderHash: header.TrustedCelestiaHeaderHash,
		TrustedRollupStateRoot:    trustedConsensusState.StateRoot,
		// TODO: check if NewHeight is the same as GetHeight
		NewHeight:             header.NewHeight,
		NewRollupStateRoot:    header.NewStateRoot,
		NewCelestiaHeaderHash: header.NewCelestiaHeaderHash,
		CodeCommitment:        cs.CodeCommitment,
		GenesisStateRoot:      cs.GenesisStateRoot,
	}

	witness, err := publicWitness.Generate()
	if err != nil {
		panic(fmt.Sprintf("failed to generate state transition public witness: %s", err))
	}

	proof := groth16.NewProof(ecc.BN254)
	_, err = proof.ReadFrom(bytes.NewReader(header.StateTransitionProof))
	if err != nil {
		panic(fmt.Sprintf("failed to read proof: %s", err))
	}

	err = groth16.Verify(proof, vk, witness)
	if err != nil {
		panic(fmt.Sprintf("failed to verify proof: %s", err))
	}

	// Check the earliest consensus state to see if it is expired, if so then set the prune height
	// so that we can delete consensus state and all associated metadata.
	var (
		pruneHeight exported.Height
		pruneError  error
	)
	pruneCb := func(height exported.Height) bool {
		consState, err := GetConsensusState(clientStore, cdc, height)
		// this error should never occur
		if err != nil {
			pruneError = err
			return true
		}
		if consState.IsExpired(sdkCtx.BlockTime()) {
			pruneHeight = height
		}
		return true
	}
	err = IterateConsensusStateAscending(clientStore, pruneCb)
	if err != nil {
		panic(fmt.Sprintf("failed to iterate consensus states: %s", err))
	}
	if pruneError != nil {
		panic(fmt.Sprintf("failed to prune consensus state: %s", pruneError))
	}
	// if pruneHeight is set, delete consensus state and metadata
	if pruneHeight != nil {
		deleteConsensusState(clientStore, pruneHeight)
		deleteConsensusMetadata(clientStore, pruneHeight)
	}

	// newClientState, consensusState := update(sdkCtx, clientStore, &cs, header)
	newConsensusState := &ConsensusState{
		HeaderTimestamp: timestamppb.New(sdkCtx.BlockTime()), // this should really be the Celestia block time at the newHeight
		StateRoot:       header.NewStateRoot,
	}

	// Q: do we need to set client state with updated height?
	setClientState(clientStore, cdc, cs)
	// set consensus state in client store
	SetConsensusState(clientStore, cdc, newConsensusState, header.GetHeight())
	// set metadata for this consensus state
	setConsensusMetadata(ctx, clientStore, header.GetHeight())
	height, ok := header.GetHeight().(clienttypes.Height)
	if !ok {
		panic(fmt.Sprintf("invalid height type %T", header.GetHeight()))
	}

	return []exported.Height{height}
}
