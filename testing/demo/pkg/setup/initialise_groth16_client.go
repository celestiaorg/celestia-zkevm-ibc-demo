package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	"github.com/ethereum/go-ethereum/ethclient"
)

// relayer address registered in simapp
const Relayer = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"

var Groth16ClientId string

func InitializeGroth16LightClientOnSimapp() error {
	fmt.Println("Initializing the groth16 light client")

	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum client: %v", err)
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

	if err := createClientOnSimapp(clientCtx, clientState, consensusState); err != nil {
		return err
	}
	fmt.Println("Groth16 client created successfully on simapp")

	if err := createChannelOnSimapp(clientCtx); err != nil {
		return err
	}
	fmt.Println("Channel created successfully on simapp")

	return nil
}

func createClientAndConsensusState(genesisBlock, latestBlock *ethtypes.Block) (*cdctypes.Any, *cdctypes.Any, error) {
	clientState := groth16.NewClientState(latestBlock.Number().Uint64(), []byte{}, []byte{}, []byte{}, genesisBlock.Root().Bytes())
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

func createClientOnSimapp(clientCtx client.Context, clientState, consensusState *cdctypes.Any) error {
	createClientMsg := &clienttypes.MsgCreateClient{
		ClientState:    clientState,
		ConsensusState: consensusState,
		Signer:         Relayer,
	}

	if clientState.GetCachedValue().(*groth16.ClientState).ClientType() != consensusState.GetCachedValue().(*groth16.ConsensusState).ClientType() {
		fmt.Println("Client and consensus state client types do not match")
	}

	createClientMsgResponse, err := utils.BroadcastMessages(clientCtx, Relayer, 200_000, createClientMsg)
	if err != nil {
		return fmt.Errorf("failed to broadcast the initial client creation message: %v", err)
	}

	if createClientMsgResponse.Code != 0 {
		return fmt.Errorf("failed to create client: %v", createClientMsgResponse.RawLog)
	}

	Groth16ClientId, err = ParseClientIDFromEvents(createClientMsgResponse.Events)
	if err != nil {
		return fmt.Errorf("failed to parse client id from events: %v", err)
	}

	return nil
}

func createChannelOnSimapp(clientCtx client.Context) error {
	cosmosMerklePathPrefix := commitmenttypesv2.NewMerklePath([]byte("simd"))
	msgCreateChannelResp, err := utils.BroadcastMessages(clientCtx, Relayer, 200_000, &channeltypesv2.MsgCreateChannel{
		ClientId:         Groth16ClientId,
		MerklePathPrefix: cosmosMerklePathPrefix,
		Signer:           Relayer,
	})
	if err != nil {
		return fmt.Errorf("failed to create channel: %v", err)
	}
	if msgCreateChannelResp.Code != 0 {
		return fmt.Errorf("failed to create channel: %v", msgCreateChannelResp.RawLog)
	}

	return nil
}

// ParseClientIDFromEvents parses events emitted from a MsgCreateClient and returns the
// client identifier.
func ParseClientIDFromEvents(events []abci.Event) (string, error) {
	for _, ev := range events {
		if ev.Type == clienttypes.EventTypeCreateClient {
			if attribute, found := attributeByKey(ev.Attributes, clienttypes.AttributeKeyClientID); found {
				return attribute.Value, nil
			}
		}
	}
	return "", errors.New("client identifier event attribute not found")
}

// attributeByKey returns the event attribute's value keyed by the given key and a boolean indicating its presence in the given attributes.
func attributeByKey(attributes []abci.EventAttribute, key string) (abci.EventAttribute, bool) {
	idx := slices.IndexFunc(attributes, func(a abci.EventAttribute) bool { return a.Key == key })
	if idx == -1 {
		return abci.EventAttribute{}, false
	}
	return attributes[idx], true
}
