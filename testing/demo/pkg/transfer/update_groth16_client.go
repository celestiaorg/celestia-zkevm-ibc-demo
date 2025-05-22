package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	// gnark "github.com/consensys/gnark/backend/groth16"
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
		return fmt.Errorf("failed to broadcast create client msg: %w", err)
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

	var reconstructedHeaderHashes = make([][32]byte, celestiaHeaderHashesLength)
	var currentIndex int
	currentIndex = 64 + 8 // first two fixed length hashes and length bytes
	for i := 0; uint64(i) < celestiaHeaderHashesLength; i++ {
		reconstructedHeaderHashes[i] = [32]byte(data[currentIndex : currentIndex+32])
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
	CelestiaHeaderHashes [][32]byte
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
	// var resp *proverclient.ProveStateTransitionResponse
	// fmt.Printf("PROOF: %v", resp.Proof)
	// fmt.Println("Public values: ", resp.GetPublicValues())
	// resp.Proof = []byte{
	// 	17, 182, 160, 157, 36, 63, 246, 147, 153, 41, 58, 13, 104, 199, 185, 205, 77, 193, 130, 20,
	// 	111, 137, 165, 234, 229, 100, 86, 211, 137, 139, 222, 224, 183, 7, 203, 131, 21, 31, 12, 239,
	// 	63, 110, 36, 69, 84, 223, 9, 90, 23, 172, 245, 90, 130, 25, 241, 104, 252, 94, 91, 98, 115,
	// 	195, 143, 57, 156, 166, 11, 97, 24, 198, 244, 118, 22, 208, 81, 127, 227, 224, 81, 182, 55,
	// 	186, 124, 78, 191, 83, 47, 211, 243, 137, 77, 180, 251, 42, 86, 222, 118, 201, 193, 137, 46,
	// 	76, 1, 195, 148, 62, 197, 56, 53, 124, 239, 75, 25, 198, 32, 27, 31, 35, 102, 196, 117, 222,
	// 	220, 170, 255, 181, 40, 30, 190, 106, 22, 12, 31, 211, 180, 154, 58, 127, 0, 129, 168, 205,
	// 	0, 157, 57, 21, 46, 164, 212, 213, 246, 52, 213, 74, 249, 85, 87, 118, 18, 243, 238, 110, 135,
	// 	176, 23, 70, 109, 4, 195, 106, 174, 63, 43, 33, 102, 62, 171, 191, 173, 19, 204, 230, 173, 182,
	// 	248, 32, 66, 12, 218, 166, 42, 251, 50, 81, 245, 90, 15, 43, 89, 187, 198, 148, 182, 56, 88,
	// 	21, 193, 192, 230, 162, 71, 38, 244, 134, 194, 94, 120, 169, 14, 238, 48, 54, 140, 166, 212,
	// 	193, 187, 209, 11, 174, 50, 182, 84, 229, 78, 150, 125, 125, 192, 14, 74, 234, 110, 18, 44,
	// 	165, 137, 195, 195, 149, 26, 137, 39, 49, 239, 146, 178, 46, 87, 137,
	// }

	// resp.PublicValues = []byte{
	// 	210, 11, 36, 149, 29, 7, 102, 236, 175, 139, 159, 217, 228, 211, 101, 12, 202, 192, 180, 162,
	// 	154, 67, 81, 4, 41, 185, 31, 106, 223, 99, 201, 174, 114, 48, 200, 21, 144, 28, 73, 93,
	// 	103, 151, 25, 104, 226, 226, 12, 106, 52, 215, 108, 90, 0, 83, 108, 225, 176, 162, 39, 123,
	// 	198, 121, 31, 234, 9, 0, 0, 0, 0, 0, 0, 0, 53, 215, 138, 14, 84, 18, 108, 31,
	// 	148, 13, 118, 102, 92, 77, 225, 177, 3, 92, 32, 140, 5, 147, 53, 94, 25, 137, 195, 26,
	// 	51, 41, 237, 51, 169, 177, 127, 164, 158, 104, 92, 122, 206, 39, 105, 88, 137, 105, 142, 84,
	// 	230, 223, 220, 169, 253, 194, 44, 12, 105, 1, 253, 236, 132, 158, 69, 15, 145, 240, 53, 150,
	// 	233, 236, 177, 184, 223, 65, 124, 53, 132, 173, 12, 165, 117, 90, 48, 207, 172, 193, 107, 214,
	// 	192, 57, 230, 175, 249, 164, 73, 40, 202, 29, 171, 86, 67, 249, 129, 152, 61, 84, 177, 26,
	// 	63, 42, 194, 14, 196, 118, 176, 4, 68, 29, 44, 61, 65, 180, 28, 238, 17, 172, 79, 208,
	// 	229, 179, 44, 242, 179, 113, 160, 171, 20, 136, 103, 210, 221, 190, 63, 22, 62, 11, 164, 145,
	// 	52, 85, 108, 227, 128, 27, 170, 89, 79, 138, 109, 203, 214, 51, 210, 59, 160, 201, 161, 134,
	// 	14, 174, 135, 129, 186, 219, 17, 67, 76, 77, 11, 251, 141, 164, 34, 191, 64, 249, 86, 12,
	// 	122, 160, 69, 161, 18, 240, 32, 55, 33, 255, 30, 28, 168, 237, 127, 131, 67, 30, 203, 162,
	// 	246, 102, 185, 206, 7, 81, 216, 172, 210, 45, 241, 28, 209, 167, 67, 216, 128, 13, 96, 174,
	// 	167, 90, 76, 80, 142, 64, 75, 233, 100, 111, 54, 44, 48, 208, 87, 79, 216, 50, 159, 184,
	// 	90, 216, 196, 139, 137, 202, 187, 150, 152, 59, 160, 195, 1, 56, 179, 11, 81, 120, 151, 12,
	// 	130, 163, 115, 250, 41, 57, 103, 168, 183, 120, 207, 151, 169, 231, 20, 170, 143, 158, 152, 103,
	// 	125, 149, 17, 94, 119, 228, 242, 64, 151, 43, 122, 204, 251, 21, 87, 47, 78, 143, 71, 220,
	// 	47, 193, 132, 27, 4, 150, 118, 185, 125, 96, 152, 97, 10, 0, 0, 0, 0, 0, 0, 0,
	// }

	// dir, err := os.Getwd()
	// buf := bytes.NewBuffer(nil)
	// vkFile, err := os.Open(dir + "/ibc/lightclients/groth16/groth16_vk.bin")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to open vk file %w", err)
	// }
	// buf.ReadFrom(vkFile)
	// vkeyBytes := buf.Bytes()

	// conn, err := grpc.NewClient(evmProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to connect to prover: %w", err)
	// }
	// defer conn.Close()
	// client := proverclient.NewProverClient(conn)

	// evmProverInfo, err := getEvmProverInfo()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get evm prover info: %w", err)
	// }

	// verifyProofRequest := &proverclient.VerifyProofRequest{
	// 	Proof:           resp.Proof,
	// 	Sp1PublicInputs: resp.GetPublicValues(),
	// 	Sp1VkeyHash:     evmProverInfo.StateTransitionVerifierKey,
	// 	Groth16Vk:       vkeyBytes,
	// }

	// fmt.Println("VERIFYING STATE TRANSITION PROOF....")
	// evmproofresp, err := client.VerifyProof(context.Background(), verifyProofRequest)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to verify state transition proof: %w", err)
	// }
	// if !evmproofresp.Success {
	// 	return nil, fmt.Errorf("failed to verify state transition proof: %w", err)
	// }
	// fmt.Println(evmproofresp)
	// fmt.Println("STATE TRANSITION PROOF VERIFIED")

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
		TrustedHeight:        trustedHeight,
		NewestHeaderHash:     blevmPublicOutput.NewestStateRoot[:],
		OldestHeaderHash:     blevmPublicOutput.OldestHeaderHash[:],
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
	// publicValues := resp.GetPublicValues()
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
