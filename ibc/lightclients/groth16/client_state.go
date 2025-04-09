package groth16

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/mpt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
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
func (cs *ClientState) status(_ sdktypes.Context, _ storetypes.KVStore, _ codec.BinaryCodec) exported.Status {
	return exported.Active
}

// Validate is a no-op.
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

func (cs *ClientState) initialize(ctx sdktypes.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, initialConsensusState exported.ConsensusState) error {
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
	_ sdktypes.Context,
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
	_ sdktypes.Context,
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
func (cs *ClientState) CheckForMisbehaviour(ctx sdktypes.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, msg exported.ClientMessage) bool {
	return false
}

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx sdktypes.Context, cdc codec.BinaryCodec, subjectClientStore, substituteClientStore storetypes.KVStore, substituteClient exported.ClientState) error {
	return sdkerrors.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update groth16 client with a proposal")
}

// VerifyUpgradeAndUpdateState returns an error because it hasn't been
// implemented yet.
func (cs *ClientState) VerifyUpgradeAndUpdateState(
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

// VerifyClientMessage checks if the clientMessage is of type Header
func (cs *ClientState) VerifyClientMessage(ctx sdktypes.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, clientStore, cdc, msg)
	default:
		return clienttypes.ErrInvalidClientType
	}
}

func (cs *ClientState) verifyHeader(_ sdktypes.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, header *Header) error {
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

// UpdateConsensusState updates the consensus state.
func (cs *ClientState) UpdateConsensusState(ctx sdktypes.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) ([]exported.Height, error) {
	header, ok := clientMsg.(*Header)
	if !ok {
		return []exported.Height{}, fmt.Errorf("the only supported clientMsg type is Header")
	}
	height, ok := header.GetHeight().(clienttypes.Height)
	if !ok {
		return []exported.Height{}, fmt.Errorf("invalid height type %T", header.GetHeight())
	}

	// Check if the consensus state has already been updated to this header.
	consensusState, err := GetConsensusState(clientStore, cdc, header.GetHeight())
	if err != nil {
		return []exported.Height{}, fmt.Errorf("failed to retrieve consensus state: %w", err)
	}
	if consensusState != nil {
		// state has already been updated to this header so this is a no-op
		return []exported.Height{header.GetHeight()}, nil
	}

	// Check if this is a mock proof (all zeros)
	isMockProof := bytes.Count(header.StateTransitionProof, []byte{0}) == len(header.StateTransitionProof)

	// If this is a mock proof, we don't need to verify it.
	if !isMockProof {
		// This is a real proof, so we need to verify it.
		trustedConsensusState, err := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, uint64(header.TrustedHeight)))
		if err != nil {
			return []exported.Height{}, fmt.Errorf("failed to get trusted consensus state: %w", err)
		}
		vk, err := DeserializeVerifyingKey(cs.StateTransitionVerifierKey)
		if err != nil {
			return []exported.Height{}, fmt.Errorf("failed to deserialize verifying key: %w", err)
		}

		publicWitness := PublicWitness{
			TrustedHeight:             header.TrustedHeight,
			TrustedCelestiaHeaderHash: header.TrustedCelestiaHeaderHash,
			TrustedRollupStateRoot:    trustedConsensusState.StateRoot,
			NewHeight:                 header.NewHeight,
			NewRollupStateRoot:        header.NewStateRoot,
			NewCelestiaHeaderHash:     header.NewCelestiaHeaderHash,
			CodeCommitment:            cs.CodeCommitment,
			GenesisStateRoot:          cs.GenesisStateRoot,
		}

		witness, err := publicWitness.Generate()
		if err != nil {
			return []exported.Height{}, fmt.Errorf("failed to generate state transition public witness: %w", err)
		}

		proof := groth16.NewProof(ecc.BN254)
		_, err = proof.ReadFrom(bytes.NewReader(header.StateTransitionProof))
		if err != nil {
			return []exported.Height{}, fmt.Errorf("failed to read proof: %w", err)
		}

		err = groth16.Verify(proof, vk, witness)
		if err != nil {
			return []exported.Height{}, fmt.Errorf("failed to verify proof: %w", err)
		}
	}

	newConsensusState := &ConsensusState{
		HeaderTimestamp: header.Timestamp,
		StateRoot:       header.NewStateRoot,
	}

	SetConsensusState(clientStore, cdc, newConsensusState, header.GetHeight())
	setConsensusMetadata(ctx, clientStore, header.GetHeight())

	return []exported.Height{height}, nil
}
