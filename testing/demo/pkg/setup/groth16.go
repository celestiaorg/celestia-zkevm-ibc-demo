package main

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CreateGroth16LightClient creates the Groth16 light client on simapp.
func CreateGroth16LightClient() error {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}
	clientState, consensusState, err := createClientAndConsensusState()
	if err != nil {
		return err
	}

	fmt.Println("Creating the Groth16 light client on simapp...")
	createClientMsgResponse, err := utils.BroadcastMessages(clientCtx, relayer, 200_000, &clienttypes.MsgCreateClient{
		ClientState:    clientState,
		ConsensusState: consensusState,
		Signer:         relayer,
	})
	if err != nil {
		return fmt.Errorf("failed to create Groth16 light client on simapp: %v", err)
	}
	if createClientMsgResponse.Code != 0 {
		return fmt.Errorf("failed to create Groth16 light client on simapp: %v", createClientMsgResponse.RawLog)
	}

	clientId, err := parseClientIDFromEvents(createClientMsgResponse.Events)
	if err != nil {
		return fmt.Errorf("failed to parse client id from events: %v", err)
	}
	fmt.Printf("Created Groth16 light client on simapp with clientId %v.\n", clientId)

	return nil
}

func createClientAndConsensusState() (*cdctypes.Any, *cdctypes.Any, error) {
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	genesisBlock, latestBlock, err := getGenesisAndLatestBlock(ethClient)
	if err != nil {
		return nil, nil, err
	}

	info, err := getEvmProverInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get evm prover info %w", err)
	}

	stateTransitionVerifierKey := info.StateTransitionVerifierKey
	fmt.Printf("Got state transition verifier key: %x\n", stateTransitionVerifierKey)

	// TODO: Uncomment this code once the EVM prover info endpoint includes a state memberhsip verifier key.
	// stateMembershipVerifierKey, err := getStateMembershipVerifierKey()
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("failed to get state membership verifier key: %w", err)
	// }
	// fmt.Printf("State state membership verifier key: %v\n", stateMembershipVerifierKey)
	stateMembershipVerifierKey := []byte{}

	// TODO: Query the codeCommitment from the EVM rollup.
	codeCommitment := []byte{}

	groth16Vk, err := getGroth16Vk()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get groth16 vk: %w", err)
	}

	clientState := groth16.NewClientState(
		latestBlock.Number().Uint64(),
		info.StateTransitionVerifierKey,
		stateMembershipVerifierKey,
		groth16Vk,
		codeCommitment,
		genesisBlock.Root().Bytes(),
	)
	clientStateAny, err := cdctypes.NewAnyWithValue(clientState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client state any: %v", err)
	}

	latestBlockTime := time.Unix(int64(latestBlock.Time()), 0)
	consensusState := groth16.NewConsensusState(latestBlockTime, latestBlock.Root().Bytes())
	consensusStateAny, err := cdctypes.NewAnyWithValue(consensusState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create consensus state any: %v", err)
	}

	if clientState.ClientType() != consensusState.ClientType() {
		fmt.Println("Client and consensus state client types do not match")
	}

	return clientStateAny, consensusStateAny, nil
}

func getGenesisAndLatestBlock(ethClient *ethclient.Client) (*ethtypes.Block, *ethtypes.Block, error) {
	genesisBlock, err := ethClient.BlockByNumber(context.Background(), big.NewInt(0))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get genesis block: %v", err)
	}

	// Keep querying for the latest block until we get one with height > 0
	var latestBlock *ethtypes.Block
	maxRetries := 30
	retryCount := 0
	retryDelay := time.Second * 5

	for {
		latestBlock, err = ethClient.BlockByNumber(context.Background(), nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get latest block: %v", err)
		}

		if latestBlock.Number().Uint64() > 0 {
			// We found a non-zero height block
			break
		}

		retryCount++
		if retryCount >= maxRetries {
			return nil, nil, fmt.Errorf("timed out waiting for a block with height > 0 after %d attempts", maxRetries)
		}

		fmt.Printf("Latest block is still genesis block (height=0), waiting %v and retrying... (attempt %d/%d)\n",
			retryDelay, retryCount, maxRetries)
		time.Sleep(retryDelay)
	}

	return genesisBlock, latestBlock, nil
}

// TODO: Uncomment this function once the EVM prover info endpoint includes a state membership verifier key.
// func getStateMembershipVerifierKey() ([]byte, error) {
// 	info, err := getEvmProverInfo()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get evm prover info %w", err)
// 	}

// 	decoded, err := hex.DecodeString(strings.TrimPrefix(info.StateMembershipVerifierKey, "0x"))
// 	if err != nil {
// 		return []byte{}, fmt.Errorf("failed to decode state membership verifier key %w", err)
// 	}
// 	return decoded, nil
// }

func getEvmProverInfo() (*proverclient.InfoResponse, error) {
	conn, err := grpc.NewClient(evmProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to evm prover: %w", err)
	}
	defer conn.Close()
	proverClient := proverclient.NewProverClient(conn)
	info, err := proverClient.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get evm prover info %w", err)
	}
	return info, nil
}

func getGroth16Vk() ([]byte, error) {
	dir, err := os.Getwd()
	buf := bytes.NewBuffer(nil)
	vkFile, err := os.Open(dir + "/ibc/lightclients/groth16/groth16_vk.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to open vk file %w", err)
	}
	buf.ReadFrom(vkFile)
	return buf.Bytes(), nil
}
