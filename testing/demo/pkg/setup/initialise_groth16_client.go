package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	"github.com/ethereum/go-ethereum/ethclient"
)

// relayer is the address registered on simapp
const relayer = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"

func InitializeGroth16LightClientOnSimapp() error {
	fmt.Println("Initializing the Groth16 light client on simapp...")

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

	if err := createChannelOnSimapp(clientCtx, clientId); err != nil {
		return err
	}

	return nil
}

func createClientAndConsensusState(genesisBlock, latestBlock *ethtypes.Block) (*cdctypes.Any, *cdctypes.Any, error) {
	// Query the info endpoint for the state transition verifier key
	// Membership proofs are currently not supported
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Getting celestia prover info...\n")
	proverClient := proverclient.NewProverClient(conn)
	info, err := proverClient.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get evm prover info %w", err)
	}
	fmt.Printf("Got evm prover info. StateTransitionVerifierKey: %v", info.StateTransitionVerifierKey)
	verifierKeyDecoded, err := hex.DecodeString(strings.TrimPrefix(info.StateTransitionVerifierKey, "0x"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode verifier key %w", err)
	}
	// Q: Do we still need the fixed size of 32 bytes considering the verifier key is of type []byte?
	// var verifierKey [32]byte
	// copy(verifierKey[:], verifierKeyDecoded)

	clientState := groth16.NewClientState(latestBlock.Number().Uint64(), verifierKeyDecoded, []byte{}, []byte{}, genesisBlock.Root().Bytes())
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

	latestBlock, err := ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest block: %v", err)
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

	fmt.Printf("Created Groth16 client with clientId %v on simapp.\n", clientId)
	return clientId, nil
}

func createChannelOnSimapp(clientCtx client.Context, clientId string) error {
	cosmosMerklePathPrefix := commitmenttypesv2.NewMerklePath([]byte("simd"))
	msg := channeltypesv2.MsgCreateChannel{
		ClientId:         clientId,
		MerklePathPrefix: cosmosMerklePathPrefix,
		Signer:           relayer,
	}
	response, err := utils.BroadcastMessages(clientCtx, relayer, 200_000, &msg)
	if err != nil {
		return fmt.Errorf("failed to create channel: %v", err)
	}
	if response.Code != 0 {
		return fmt.Errorf("failed to create channel: %v", response.RawLog)
	}

	fmt.Println("Created channel on simapp.")
	return nil
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
