package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
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
	fmt.Println(resp, "PROOF RESPONSE")

	dir, err := os.Getwd()
	buf := bytes.NewBuffer(nil)
	vkFile, err := os.Open(dir + "/ibc/lightclients/groth16/groth16_vk.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to open vk file %w", err)
	}
	buf.ReadFrom(vkFile)
	vkeyBytes := buf.Bytes()

	conn, err := grpc.NewClient(evmProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()
	client := proverclient.NewProverClient(conn)


	evmProverInfo, err := getEvmProverInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get evm prover info: %w", err)
	}

	verifyProofRequest := &proverclient.VerifyProofRequest{
		Proof:           resp.GetProof(),
		Sp1PublicInputs: resp.GetPublicValues(),
		Sp1VkeyHash:     evmProverInfo.StateTransitionVerifierKey,
		Groth16Vk:       vkeyBytes,
	}

	fmt.Println("VERIFYING STATE TRANSITION PROOF....")
	evmproofresp, err := client.VerifyProof(context.Background(), verifyProofRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get state transition proof: %w", err)
	}
	fmt.Println(evmproofresp)
	fmt.Println("STATE TRANSITION PROOF VERIFIED")

	// publicValuesPassed := []byte{
	// 	// Number of celestia header hashes (11)
	// 	11, 0, 0, 0, 0, 0, 0, 0,

	// 	// Newest header hash
	// 	130, 145, 146, 189, 207, 51, 223, 212, 130, 247, 223, 62, 65, 247, 138, 53, 31, 95, 170, 173, 170, 236, 62, 159, 130, 119, 143, 80, 106, 167, 179, 22,

	// 	// Oldest header hash
	// 	192, 160, 132, 95, 74, 173, 130, 202, 126, 212, 70, 86, 85, 143, 210, 219, 89, 49, 82, 157, 181, 31, 251, 70, 16, 142, 6, 45, 220, 33, 10, 160,

	// 	// Celestia header hashes (11 hashes)
	// 	95, 138, 224, 134, 39, 215, 175, 19, 32, 250, 40, 109, 36, 247, 105, 227, 161, 116, 139, 86, 93, 16, 150, 118, 125, 134, 74, 189, 151, 160, 245, 225,
	// 	198, 93, 32, 141, 113, 80, 76, 21, 135, 241, 141, 169, 228, 162, 152, 254, 108, 50, 154, 206, 142, 171, 26, 20, 246, 72, 206, 18, 231, 241, 210, 7,
	// 	160, 249, 255, 241, 214, 124, 123, 20, 64, 160, 98, 226, 38, 85, 182, 33, 72, 223, 192, 179, 235, 220, 174, 3, 82, 241, 225, 149, 171, 239, 198, 123,
	// 	203, 191, 95, 211, 74, 111, 114, 241, 238, 86, 41, 110, 6, 65, 89, 19, 0, 61, 113, 161, 156, 146, 161, 255, 241, 93, 24, 53, 61, 132, 219, 252,
	// 	51, 187, 189, 127, 91, 94, 209, 149, 218, 80, 11, 137, 171, 128, 52, 55, 18, 182, 95, 184, 254, 61, 50, 170, 65, 107, 231, 72, 176, 133, 230, 102,
	// 	127, 146, 195, 230, 17, 220, 173, 252, 239, 118, 2, 127, 3, 73, 125, 68, 22, 93, 41, 19, 21, 156, 178, 23, 132, 42, 54, 97, 204, 90, 124, 42,
	// 	64, 239, 203, 114, 217, 229, 27, 20, 155, 197, 96, 10, 40, 179, 109, 58, 252, 176, 7, 68, 54, 42, 19, 168, 174, 57, 167, 202, 149, 188, 202, 53,
	// 	191, 203, 115, 196, 186, 43, 127, 51, 72, 228, 93, 213, 198, 150, 234, 233, 45, 55, 1, 216, 72, 83, 251, 8, 66, 5, 8, 65, 169, 178, 52, 26,
	// 	120, 138, 228, 142, 116, 74, 7, 247, 116, 52, 82, 189, 193, 78, 4, 49, 107, 62, 211, 62, 27, 103, 102, 2, 204, 163, 111, 242, 107, 241, 72, 75,
	// 	2, 239, 63, 95, 199, 91, 121, 42, 80, 6, 140, 92, 133, 174, 244, 235, 17, 107, 82, 66, 136, 220, 208, 161, 78, 191, 162, 239, 250, 127, 41, 68,
	// 	8, 71, 227, 148, 169, 10, 110, 208, 158, 243, 139, 158, 122, 80, 240, 44, 87, 7, 227, 251, 121, 148, 123, 94, 206, 229, 239, 72, 209, 51, 33, 39,

	// 	// Newest state root
	// 	70, 228, 88, 200, 88, 88, 169, 112, 127, 189, 121, 209, 101, 253, 163, 171, 148, 78, 60, 154, 46, 62, 208, 88, 95, 66, 108, 21, 181, 82, 58, 207,

	// 	// Newest height (15)
	// 	15, 0, 0, 0, 0, 0, 0, 0,
	// }
	// trustedHeight, err := getTrustedHeight()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get trusted height: %w", err)
	// }

	// newStateRoot, newHeight, timestamp, err := getEVMStateRootHeightTimestamp(evmTransferBlockNumber)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get EVM state root, height, and timestamp: %w", err)
	// }
	// publicValues, err := DecodePublicValues(publicValuesPassed)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to decode public values: %w", err)
	// }
	// fmt.Printf("Public values: %v\n", publicValues)

	// oldestHeaderHash, newestHeaderHash, err := getFirstAndLastHeaderHashes(evmTransferBlockNumber)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get first and last header hashes: %w", err)
	// }

	// stateTransitionProof = []byte{}
	// trustedHeight = 0
	// newHeight = 0
	// newStateRoot = []byte{}
	// oldestHeaderHash := []byte{}
	// timestamp = time.Now()
	// // oldestHeaderHash := []byte{}
	// // newestHeaderHash := []byte{}
	// // HARD CODE ALL VALUES
	// header := &groth16Client.Header{
	// 	StateTransitionProof: stateTransitionProof,
	// 	TrustedHeight:        trustedHeight,
	// 	NewestHeaderHash:     newStateRoot,
	// 	OldestHeaderHash:     oldestHeaderHash,
	// 	NewestStateRoot:      newStateRoot,
	// }

	// TEST PROOF VERIFICATION
	// vk, err := groth16Client.DeserializeVerifyingKey(stateTransitionProof)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to deserialize verifying key: %w", err)
	// }

	// publicWitness := groth16Client.PublicWitness{
	// 	NewestHeaderHash:     header.NewestHeaderHash,
	// 	OldestHeaderHash:     header.OldestHeaderHash,
	// 	CelestiaHeaderHashes: header.CelestiaHeaderHashes,
	// 	NewestStateRoot:      header.NewestStateRoot,
	// 	NewestHeight:         header.NewestHeight,
	// }

	// witness, err := publicWitness.Generate()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to generate state transition public witness: %w", err)
	// }

	// proof := gnark.NewProof(ecc.BN254)
	// _, err = proof.ReadFrom(bytes.NewReader(header.StateTransitionProof))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to read proof: %w", err)
	// }

	// fmt.Printf("Verifying state transition proof...\n")
	// err = gnark.Verify(proof, vk, witness)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to verify proof: %w", err)
	// }

	// get oldest header hash

	// header := &groth16Client.Header{
	// 	StateTransitionProof: stateTransitionProof,
	// 	TrustedHeight:        trustedHeight,
	// 	NewestHeaderHash:     newStateRoot,
	// 	OldestHeaderHash:     oldestHeaderHash,
	// 	NewestStateRoot:      newStateRoot,
	// 	NewestHeight:         uint64(newHeight),
	// 	Timestamp:            timestamppb.New(timestamp),
	// }

	return &groth16Client.Header{}, nil
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

func getEVMStateRootHeightTimestamp(evmTransferBlockNumber uint64) ([]byte, int64, time.Time, error) {
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, 0, time.Time{}, fmt.Errorf("failed to connect to Reth: %w", err)
	}

	header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(evmTransferBlockNumber)))
	if err != nil {
		return nil, 0, time.Time{}, fmt.Errorf("failed to get latest header: %w", err)
	}

	return header.Root.Bytes(), header.Number.Int64(), time.Unix(int64(header.Time), 0), nil
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
