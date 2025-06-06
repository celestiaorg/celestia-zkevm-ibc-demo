package groth16

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/mpt"
	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	Groth16ClientType = ModuleName
)

// MptProof contains the Merkle Patricia Trie proofs for packet commitment verification.
// It includes both account and storage proofs from eth_getProof response.
// Ref: https://eips.ethereum.org/EIPS/eip-1186
type MptProof struct {
	// The account proof is used to verify the account state of the ICS26Router contract.
	AccountProof []hexutil.Bytes `json:"accountProof"`
	Address      common.Address  `json:"address"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`

	// Storage proof for the packet commitment in ICS26Router contract storage.
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []hexutil.Bytes `json:"storageProof"`
	StorageKey   common.Hash     `json:"storageKey"`
	StorageValue hexutil.Big     `json:"storageValue"`
}

// ClientState implements the exported.ClientState interface for Groth16 light clients.
var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance.
func NewClientState(latestHeight uint64, stateTransitionVerifierKey string, stateMembershipVerifierKey []byte, groth16Vk []byte, codeCommitment []byte, genesisStateRoot []byte) *ClientState {
	return &ClientState{
		LatestHeight:               latestHeight,
		CodeCommitment:             codeCommitment,
		GenesisStateRoot:           genesisStateRoot,
		StateTransitionVerifierKey: stateTransitionVerifierKey,
		StateMembershipVerifierKey: stateMembershipVerifierKey,
		Groth16Vk:                  groth16Vk,
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
	_ exported.Path,
	value []byte,
) error {
	// Get consensus state for verification
	consensusState, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}

	// Deserialize the MPT proof
	var deserializedProof MptProof
	if err := json.Unmarshal(proof, &deserializedProof); err != nil {
		return fmt.Errorf("failed to deserialize mpt proof: %w", err)
	}

	// Verify ICS26Router contract account exists in state
	ICS26RouterAddress := crypto.Keccak256(deserializedProof.Address.Bytes())
	verifiedAccountState, err := mpt.VerifyMerklePatriciaTrieProof(
		ethcommon.BytesToHash(consensusState.StateRoot),
		ICS26RouterAddress,
		deserializedProof.AccountProof,
	)
	if err != nil {
		return fmt.Errorf("inclusion verification failed: %w", err)
	}

	// Reconstruct and verify account state
	accountState := []any{
		uint64(deserializedProof.Nonce),
		deserializedProof.Balance.ToInt().Bytes(),
		deserializedProof.StorageHash,
		deserializedProof.CodeHash,
	}
	encodedAccountState, err := rlp.EncodeToBytes(accountState)
	if err != nil {
		return fmt.Errorf("failed to rlp encode reconstructed account value: %w", err)
	}
	if !bytes.Equal(verifiedAccountState, encodedAccountState) {
		return fmt.Errorf("expected account claimed value: %x does not match the verified value: %x",
			encodedAccountState, verifiedAccountState)
	}

	// Verify packet commitment exists in contract storage
	commitmentPath := crypto.Keccak256(deserializedProof.StorageKey.Bytes())
	verifiedValue, err := mpt.VerifyMerklePatriciaTrieProof(
		deserializedProof.StorageHash,
		commitmentPath,
		deserializedProof.StorageProof,
	)
	if err != nil {
		return fmt.Errorf("inclusion verification failed: %w", err)
	}

	// Verify storage value matches expected
	expectedValue, err := rlp.EncodeToBytes(value)
	if err != nil {
		return fmt.Errorf("failed to encode value: %w", err)
	}
	if !bytes.Equal(verifiedValue, expectedValue) {
		return fmt.Errorf("verified value: %x does not match the expected value: %x",
			verifiedValue, expectedValue)
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

	// NOTE: this verification implementation is not correct, mpt getProof doesn't handle non-membership proofs
	mptKey := append(merklePath.KeyPath[0], merklePath.KeyPath[1]...)

	var decodedProof MptProof
	err = json.Unmarshal(proof, &decodedProof)
	if err != nil {
		return fmt.Errorf("failed to deserialize mpt proof: %w", err)
	}

	// Inclusion verification only supports MPT tries currently
	verifiedValue, err := mpt.VerifyMerklePatriciaTrieProof(ethcommon.BytesToHash(consensusState.StateRoot), mptKey, decodedProof.AccountProof)
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

// UpdateState updates the consensus state and client state.
func (cs *ClientState) UpdateState(ctx sdktypes.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) ([]exported.Height, error) {
	header, ok := clientMsg.(*Header)
	if !ok {
		return []exported.Height{}, fmt.Errorf("the only supported clientMsg type is Header")
	}

	height, ok := header.GetHeight().(clienttypes.Height)
	if !ok {
		return []exported.Height{}, fmt.Errorf("invalid height type %T", header.GetHeight())
	}

	evmProverRPC := os.Getenv("EVM_PROVER_URL")
	conn, err := grpc.NewClient(evmProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()
	client := proverclient.NewProverClient(conn)

	verifyProofRequest := &proverclient.VerifyProofRequest{
		Proof:           header.StateTransitionProof,
		Sp1PublicInputs: header.PublicValues,
		Sp1VkeyHash:     cs.StateTransitionVerifierKey,
		Groth16Vk:       cs.Groth16Vk,
	}

	fmt.Printf("Verifying groth16 state transition proof from height: %d to height: %d\n", header.GetHeight(), header.NewestHeight)
	verifyProofResponse, err := client.VerifyProof(ctx, verifyProofRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to verify proof: %w", err)
	}
	fmt.Println(verifyProofResponse, "verify proof response")

	if !verifyProofResponse.Success {
		return nil, fmt.Errorf("proof verification failed")
	}

	newConsensusState := &ConsensusState{
		HeaderTimestamp: header.Timestamp,
		StateRoot:       header.NewestStateRoot,
	}

	fmt.Printf("Setting new consensus state with state root: %X and height: %v and timestamp: %v\n", newConsensusState.StateRoot, header.GetHeight(), newConsensusState.HeaderTimestamp)
	SetConsensusState(clientStore, cdc, newConsensusState, header.GetHeight())
	setConsensusMetadata(ctx, clientStore, header.GetHeight())

	newClientState := &ClientState{
		LatestHeight:               header.GetHeight().GetRevisionHeight(),
		CodeCommitment:             cs.CodeCommitment,
		GenesisStateRoot:           cs.GenesisStateRoot,
		StateTransitionVerifierKey: cs.StateTransitionVerifierKey,
		StateMembershipVerifierKey: cs.StateMembershipVerifierKey,
		Groth16Vk:                  cs.Groth16Vk,
	}
	fmt.Printf("Setting new client state with height: %v\n", newClientState.LatestHeight)
	setClientState(clientStore, cdc, newClientState)

	return []exported.Height{height}, nil
}
