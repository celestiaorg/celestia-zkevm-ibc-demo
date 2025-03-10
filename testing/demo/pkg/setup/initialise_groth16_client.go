package main

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/ethereum/go-ethereum/ethclient"
)

// relayer is the address registered on simapp
const relayer = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"

func InitializeGroth16LightClientOnSimapp() error {
	fmt.Println("Creating the Groth16 light client on simapp...")

	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	genesisBlock, latestBlock, err := getGenesisAndLatestBlock(ethClient)
	if err != nil {
		return err
	}

	clientState, consensusState, err := createClientAndConsensusState(genesisBlock, latestBlock)
	if err != nil {
		return err
	}

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	clientId, err := createClientOnSimapp(clientCtx, clientState, consensusState)
	if err != nil {
		return err
	}
	fmt.Printf("Created Groth16 light client on simapp with clientId %v.\n", clientId)

	return nil
}

func createClientAndConsensusState(genesisBlock, latestBlock *ethtypes.Block) (*cdctypes.Any, *cdctypes.Any, error) {
	// Query the info endpoint for the state transition verifier key
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()

	// TODO: Query this from the EVM rollup.
	codeCommitment := []byte{}
	stateTransitionVerifierKey := []byte{}
	stateMembershipVerifierKey := []byte{}

	clientState := groth16.NewClientState(
		latestBlock.Number().Uint64(),
		stateTransitionVerifierKey,
		stateMembershipVerifierKey,
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

func createClientOnSimapp(clientCtx client.Context, clientState, consensusState *cdctypes.Any) (clientId string, err error) {
	createClientMsg := &clienttypes.MsgCreateClient{
		ClientState:    clientState,
		ConsensusState: consensusState,
		Signer:         relayer,
	}

	if clientState.GetCachedValue().(*groth16.ClientState).ClientType() != consensusState.GetCachedValue().(*groth16.ConsensusState).ClientType() {
		fmt.Println("Client and consensus state client types do not match")
	}

	createClientMsgResponse, err := utils.BroadcastMessages(clientCtx, relayer, 200_000, createClientMsg)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast the initial client creation message: %v", err)
	}

	if createClientMsgResponse.Code != 0 {
		return "", fmt.Errorf("failed to create client: %v", createClientMsgResponse.RawLog)
	}

	clientId, err = parseClientIDFromEvents(createClientMsgResponse.Events)
	if err != nil {
		return "", fmt.Errorf("failed to parse client id from events: %v", err)
	}

	return clientId, nil
}

// parseClientIDFromEvents parses events emitted from a MsgCreateClient and
// returns the client identifier.
func parseClientIDFromEvents(events []abci.Event) (string, error) {
	for _, event := range events {
		if event.Type == clienttypes.EventTypeCreateClient {
			if attribute, isFound := getAttributeByKey(event.Attributes, clienttypes.AttributeKeyClientID); isFound {
				return attribute.Value, nil
			}
		}
	}
	return "", fmt.Errorf("client identifier event attribute not found")
}

// getAttributeByKey returns the first event attribute with the given key.
func getAttributeByKey(attributes []abci.EventAttribute, key string) (ea abci.EventAttribute, isFound bool) {
	idx := slices.IndexFunc(attributes, func(a abci.EventAttribute) bool { return a.Key == key })
	if idx == -1 {
		return abci.EventAttribute{}, false
	}
	return attributes[idx], true
}
