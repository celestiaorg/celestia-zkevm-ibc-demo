package groth16

import (
	"bytes"
	"context"
	fmt "fmt"

	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/mpt"
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

const (
	Groth16ClientType = ModuleName
)

// ClientState implements the exported.ClientState interface for Groth16 light clients.
var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance.
func NewClientState(latestHeight uint64, stateTransitionVerifierKey []byte, stateMembershipVerifierKey []byte, codeCommitment []byte, genesisStateRoot []byte) *ClientState {
	return &ClientState{
		LatestHeight:               latestHeight,
		CodeCommitment:             codeCommitment,
		GenesisStateRoot:           genesisStateRoot,
		StateTransitionVerifierKey: stateMembershipVerifierKey,
		StateMembershipVerifierKey: stateMembershipVerifierKey,
	}
}

// ClientType returns the groth16 client type.
func (cs *ClientState) ClientType() string {
	return Groth16ClientType
}

// GetLatestClientHeight returns the latest block height of the client state.
func (cs *ClientState) GetLatestClientHeight() exported.Height {
	return clienttypes.Height{
		RevisionNumber: 0,
		RevisionHeight: cs.LatestHeight,
	}
}

// status returns the status of the groth16 client.
func (cs *ClientState) status(_ context.Context, _ storetypes.KVStore, _ codec.BinaryCodec) exported.Status {
	return exported.Active
}

func (cs *ClientState) Validate() error {
	return nil
}

// ZeroCustomFields returns a ClientState that is a copy of the current ClientState
// with all client customizable fields zeroed out.
func (cs *ClientState) ZeroCustomFields() exported.ClientState {
	// Copy over all chain-specified fields and leave custom fields empty.
	return &ClientState{
		LatestHeight:               cs.LatestHeight,
		StateTransitionVerifierKey: cs.StateTransitionVerifierKey,
	}
}

func (cs *ClientState) initialize(ctx context.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, initialConsensusState exported.ConsensusState) error {
	consensusState, ok := initialConsensusState.(*ConsensusState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T", &ConsensusState{}, initialConsensusState)
	}
	height := cs.GetLatestClientHeight()
	SetConsensusState(clientStore, cdc, consensusState, height)
	setConsensusMetadata(ctx, clientStore, height)
	setClientState(clientStore, cdc, cs)
	return nil
}

//------------------------------------

// The following are modified methods from the v9 IBC Client interface. The idea is to make
// it easy to update this client once Celestia moves to v9 of IBC
func (cs *ClientState) verifyMembership(
	_ context.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	_ uint64,
	_ uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	// Path validation
	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return sdkerrors.Wrapf(commitmenttypes.ErrInvalidProof, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	consensusState, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}

	// MPT takes keypath as []byte, so we concatenate the keys arrays
	// TODO we might have to change this because based on tests the keypath is always one element
	mptKey := merklePath.KeyPath[0]

	// Inclusion verification only supports MPT tries currently
	verifiedValue, err := mpt.VerifyMerklePatriciaTrieProof(consensusState.StateRoot, mptKey, proof)
	if err != nil {
		return fmt.Errorf("inclusion verification failed: %w", err)
	}

	if !bytes.Equal(value, verifiedValue) {
		return fmt.Errorf("retrieved value does not match the value passed to the client")
	}

	return nil
}

// verifyNonMembership verifies a proof of the absence of a key in the Merkle tree.
// It's the same as VerifyMembership, but the value is nil.
func (cs *ClientState) verifyNonMembership(
	_ context.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	_ uint64,
	_ uint64,
	proof []byte,
	path exported.Path,
) error {
	consensusState, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return sdkerrors.Wrapf(commitmenttypes.ErrInvalidProof, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	// MPT takes keypath as []byte, so we concatenate the keys arrays
	// TODO we might have to change this because based on tests the keypath is always one element
	mptKey := append(merklePath.KeyPath[0], merklePath.KeyPath[1]...)

	// Inclusion verification only supports MPT tries currently
	verifiedValue, err := mpt.VerifyMerklePatriciaTrieProof(consensusState.StateRoot, mptKey, proof)
	if err != nil {
		return fmt.Errorf("inclusion verification failed: %w", err)
	}

	// if verifiedValue is not nil error
	if verifiedValue != nil {
		return fmt.Errorf("the value for the specified key exists: %w", err)
	}

	return nil
}

func (cs *ClientState) getTimestampAtHeight(clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (uint64, error) {
	consensusState, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return 0, fmt.Errorf("failed to get consensus state: %w", err)
	}

	return consensusState.GetTimestamp(), nil
}

// CheckForMisbehaviour is a no-op for groth16
func (cs *ClientState) CheckForMisbehaviour(ctx context.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, msg exported.ClientMessage) bool {
	return false
}

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx context.Context, cdc codec.BinaryCodec, subjectClientStore, substituteClientStore storetypes.KVStore, substituteClient exported.ClientState) error {
	return sdkerrors.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update groth16 client with a proposal")
}
