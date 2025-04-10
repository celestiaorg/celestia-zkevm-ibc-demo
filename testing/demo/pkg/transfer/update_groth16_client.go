package main

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func updateGroth16LightClient() error {
	fmt.Printf("Updating Groth16 light client on EVM roll-up...\n")
	consensusState, err := getConsensusState()
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}
	fmt.Printf("Groth16 light client current timetstamp %v and state root %X\n", consensusState.GetHeaderTimestamp(), consensusState.GetStateRoot())

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to get client context: %w", err)
	}

	header, err := getHeader()
	if err != nil {
		return fmt.Errorf("failed to get header: %w", err)
	}

	clientMessage, err := cdctypes.NewAnyWithValue(header)
	if err != nil {
		return fmt.Errorf("failed to create any value: %w", err)
	}

	resp, err := utils.BroadcastMessages(clientCtx, sender, 200_000, &clienttypes.MsgUpdateClient{
		ClientId:      groth16ClientID,
		ClientMessage: clientMessage,
		Signer:        sender,
	})
	if err != nil {
		return fmt.Errorf("failed to broadcast create client msg: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("failed to update Groth16 light client on simapp: %w", err)
	}
	fmt.Printf("Updated Groth16 light client on simapp.\n")

	newConsensusState, err := getConsensusState()
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}
	fmt.Printf("Groth16 light client current timetstamp %v and state root %X\n", newConsensusState.GetHeaderTimestamp(), newConsensusState.GetStateRoot())
	return nil
}

func getHeader() (*groth16.Header, error) {
	mockProof := []byte{0}
	trustedHeight, err := getTrustedHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get trusted height: %w", err)
	}

	newStateRoot, newHeight, timestamp, err := getEVMStateRootHeightTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get EVM state root, height, and timestamp: %w", err)
	}

	header := &groth16.Header{
		StateTransitionProof:      mockProof,
		TrustedHeight:             trustedHeight,
		TrustedCelestiaHeaderHash: []byte{},
		NewStateRoot:              newStateRoot,
		NewHeight:                 newHeight,
		NewCelestiaHeaderHash:     []byte{},
		DataRoots:                 [][]byte{},
		Timestamp:                 timestamppb.New(timestamp),
	}

	fmt.Printf("Header.NewHeight: %v\n", header.NewHeight)
	fmt.Printf("Header.NewStateRoot: %X\n", header.NewStateRoot)
	fmt.Printf("Header.Timestamp: %v\n", header.Timestamp)

	return header, nil
}

// getTrustedHeight returns the last trusted height that the Groth16 light client is aware of.
func getTrustedHeight() (int64, error) {
	clientState, err := getClientState()
	if err != nil {
		return 0, fmt.Errorf("failed to get groth16 client state: %w", err)
	}

	// Get the latest height from the client state
	height := clientState.LatestHeight
	return int64(height), nil
}

func getClientState() (*groth16.ClientState, error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get client context: %w", err)
	}

	// Query the client state
	queryClient := clienttypes.NewQueryClient(clientCtx)
	resp, err := queryClient.ClientState(context.Background(), &clienttypes.QueryClientStateRequest{
		ClientId: groth16ClientID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query client state: %w", err)
	}

	// Try to unpack the client state using the exported interface
	var clientState exported.ClientState
	if err := clientCtx.InterfaceRegistry.UnpackAny(resp.ClientState, &clientState); err != nil {
		return nil, fmt.Errorf("failed to unpack client state: %w", err)
	}

	// Type assert to the Groth16 client state
	groth16ClientState, ok := clientState.(*groth16.ClientState)
	if !ok {
		return nil, fmt.Errorf("failed to type assert to Groth16 client state, got type %T", clientState)
	}

	return groth16ClientState, nil
}

func getEVMStateRootHeightTimestamp() ([]byte, int64, time.Time, error) {
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, 0, time.Time{}, fmt.Errorf("failed to connect to Reth: %w", err)
	}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, 0, time.Time{}, fmt.Errorf("failed to get latest header: %w", err)
	}

	return header.Root.Bytes(), header.Number.Int64(), time.Unix(int64(header.Time), 0), nil
}

func getConsensusState() (*groth16.ConsensusState, error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get client context: %w", err)
	}

	queryClient := clienttypes.NewQueryClient(clientCtx)
	resp, err := queryClient.ConsensusState(context.Background(), &clienttypes.QueryConsensusStateRequest{
		ClientId:     groth16ClientID,
		LatestHeight: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query consensus state: %w", err)
	}

	var consensusState exported.ConsensusState
	if err := clientCtx.InterfaceRegistry.UnpackAny(resp.ConsensusState, &consensusState); err != nil {
		return nil, fmt.Errorf("failed to unpack consensus state: %w", err)
	}

	groth16ConsensusState, ok := consensusState.(*groth16.ConsensusState)
	if !ok {
		return nil, fmt.Errorf("failed to type assert to Groth16 consensus state, got type %T", consensusState)
	}

	return groth16ConsensusState, nil
}
