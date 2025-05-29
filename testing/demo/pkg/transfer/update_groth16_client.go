package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	groth16Client "github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func updateGroth16LightClient(evmTransferBlockNumber uint64) error {
	fmt.Printf("Updating Groth16 light client on EVM roll-up...\n")

	clientState, err := getClientState()
	if err != nil {
		return fmt.Errorf("failed to get client state: %w", err)
	}
	consensusState, err := getConsensusState()
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}
	fmt.Printf("Groth16 light client current height %v and state root %X\n", clientState.LatestHeight, consensusState.GetStateRoot())

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to get client context: %w", err)
	}

	header, err := getHeader(evmTransferBlockNumber)
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
		return fmt.Errorf("failed to broadcast update client msg: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("failed to update Groth16 light client on simapp: %w", err)
	}

	newConsensusState, err := getConsensusState()
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}
	newClientState, err := getClientState()
	if err != nil {
		return fmt.Errorf("failed to get client state: %w", err)
	}
	fmt.Printf("Updated Groth16 light client on simapp. New height: %v state root %X\n", newClientState.LatestHeight, newConsensusState.GetStateRoot())

	return nil
}

func DecodePublicValues(data []byte) (*BlevmAggOutput, error) {

	// Create a new buffer with the data
	buf := bytes.NewBuffer(data)
	output := &BlevmAggOutput{}

	// Read fixed-size fields
	if err := binary.Read(buf, binary.LittleEndian, &output.NewestHeaderHash); err != nil {
		return nil, fmt.Errorf("read newest header hash: %w", err)
	}

	if err := binary.Read(buf, binary.LittleEndian, &output.OldestHeaderHash); err != nil {
		return nil, fmt.Errorf("read oldest header hash: %w", err)
	}

	celestiaHeaderHashesLength := binary.LittleEndian.Uint64(data[64 : 64+8])

	var reconstructedHeaderHashes = make([][]byte, celestiaHeaderHashesLength)
	var currentIndex int
	currentIndex = 64 + 8 // first two fixed length hashes and length bytes
	for i := 0; uint64(i) < celestiaHeaderHashesLength; i++ {
		reconstructedHeaderHashes[i] = []byte(data[currentIndex : currentIndex+32])
		if len(reconstructedHeaderHashes[i]) != 32 {
			fmt.Errorf("something wrong with celestia header hashes reconstruction")
		}
		currentIndex = currentIndex + 32
	}
	output.CelestiaHeaderHashes = reconstructedHeaderHashes

	// Read remaining fields
	output.NewestStateRoot = [32]byte(data[currentIndex : currentIndex+32])
	output.NewestHeight = binary.LittleEndian.Uint64(data[len(data)-8:])

	fmt.Println("OUTPUT", output)

	fmt.Printf("Successfully decoded public values with %d celestia header hashes\n", len(output.CelestiaHeaderHashes))
	return output, nil
}

type BlevmAggOutput struct {
	// newest_header_hash is the last block's hash on the EVM roll-up
	NewestHeaderHash [32]byte
	// oldest_header_hash is the earliest block's hash on the EVM roll-up
	OldestHeaderHash [32]byte
	// celestia_header_hashes is the range of Celestia blocks that include all
	// of the blob data the EVM roll-up has posted from oldest_header_hash to
	// newest_header_hash
	CelestiaHeaderHashes [][]byte
	// newest_state_root is the computed state root of the EVM roll-up after
	// processing blocks from oldest_header_hash to newest_header_hash
	NewestStateRoot [32]byte
	// newest_height is the most recent block number of the EVM roll-up
	NewestHeight uint64
}

func getHeader(evmTransferBlockNumber uint64) (*groth16Client.Header, error) {
	resp, err := getProof()
	if err != nil {
		return nil, fmt.Errorf("failed to get proof: %w", err)
	}

	trustedHeight, err := getTrustedHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get trusted height: %w", err)
	}

	blevmPublicOutput, err := DecodePublicValues(resp.GetPublicValues())
	if err != nil {
		return nil, fmt.Errorf("failed to decode public values: %w", err)
	}

	timestamp, err := getEVMTimestampAtHeight(evmTransferBlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get evm timestamp at height: %w", err)
	}

	header := &groth16Client.Header{
		StateTransitionProof: resp.Proof,
		PublicValues:         resp.GetPublicValues(),
		TrustedHeight:        trustedHeight,
		NewestStateRoot:      blevmPublicOutput.NewestStateRoot[:],
		NewestHeight:         blevmPublicOutput.NewestHeight,
		Timestamp:            timestamppb.New(timestamp),
	}

	return header, nil
}

// getProof queries EVM prover for a state transition proof from the last trusted height to the latest reth height.
func getProof() (*proverclient.ProveStateTransitionResponse, error) {
	conn, err := grpc.NewClient(evmProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()
	client := proverclient.NewProverClient(conn)

	fmt.Printf("Requesting evm-prover state transition proof...\n")
	resp, err := client.ProveStateTransition(context.Background(), &proverclient.ProveStateTransitionRequest{ClientId: groth16ClientID})
	if err != nil {
		return nil, fmt.Errorf("failed to get state transition proof: %w", err)
	}
	fmt.Printf("Received evm-prover state transition proof.\n")
	return resp, nil
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

func getClientState() (*groth16Client.ClientState, error) {
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
	groth16ClientState, ok := clientState.(*groth16Client.ClientState)
	if !ok {
		return nil, fmt.Errorf("failed to type assert to Groth16 client state, got type %T", clientState)
	}

	return groth16ClientState, nil
}

func getEVMTimestampAtHeight(evmTransferBlockNumber uint64) (time.Time, error) {
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to connect to Reth: %w", err)
	}

	header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(evmTransferBlockNumber)))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest header: %w", err)
	}

	return time.Unix(int64(header.Time), 0), nil
}

func getFirstAndLastHeaderHashes(evmTransferBlockNumber uint64) ([]byte, []byte, error) {
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Reth: %w", err)
	}

	firstBlock, err := client.BlockByNumber(context.Background(), big.NewInt(0))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get first block: %w", err)
	}

	lastBlock, err := client.BlockByNumber(context.Background(), big.NewInt(int64(evmTransferBlockNumber)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get last block: %w", err)
	}

	return firstBlock.Hash().Bytes(), lastBlock.Hash().Bytes(), nil
}

func getConsensusState() (*groth16Client.ConsensusState, error) {
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

	groth16ConsensusState, ok := consensusState.(*groth16Client.ConsensusState)
	if !ok {
		return nil, fmt.Errorf("failed to type assert to Groth16 consensus state, got type %T", consensusState)
	}

	return groth16ConsensusState, nil
}

func getEvmProverInfo() (*proverclient.InfoResponse, error) {
	conn, err := grpc.NewClient(evmProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()
	client := proverclient.NewProverClient(conn)

	fmt.Printf("Requesting evm-prover info...\n")
	resp, err := client.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get prover info: %w", err)
	}

	return resp, nil
}
