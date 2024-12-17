package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cosmossdk.io/x/tx/signing"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacysigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// relayer address registered in simapp
const Relayer = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"

var Groth16ClientId string

func InitializeGroth16LightClientOnSimapp() error {
	fmt.Println("Initializing the groth16 light client on the simapp...")

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

	// simapp address
	clientCtx, err := SetupClientContext()
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
		return fmt.Errorf("client and consensus state client types do not match")
	}

	createClientMsgResponse, err := BroadcastMessages(clientCtx, Relayer, 200_000, createClientMsg)
	if err != nil {
		return fmt.Errorf("failed to broadcast the initial client creation message: %w", err)
	}

	if createClientMsgResponse.Code != 0 {
		return fmt.Errorf("failed to create client: %v", createClientMsgResponse.RawLog)
	}

	Groth16ClientId, err = ParseClientIDFromEvents(createClientMsgResponse.Events)
	if err != nil {
		return fmt.Errorf("failed to parse client id from events: %w", err)
	}

	return nil
}

func createChannelOnSimapp(clientCtx client.Context) error {
	cosmosMerklePathPrefix := commitmenttypesv2.NewMerklePath([]byte("simd"))
	msgCreateChannelResp, err := BroadcastMessages(clientCtx, Relayer, 200_000, &channeltypesv2.MsgCreateChannel{
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

// SetupClientContext initializes a Cosmos SDK ClientContext
func SetupClientContext() (client.Context, error) {
	// Get the user's home directory
	home, err := os.Getwd()
	if err != nil {
		return client.Context{}, fmt.Errorf("failed to initialize keyring: %v", err)
	}

	// Chain-specific configurations
	chainID := "zkibc-demo"
	cometNodeURI := "http://localhost:5123"                                // Comet RPC endpoint
	appName := "celestia-zkevm-ibc-demo"                                   // Name of the application from the genesis file
	grpcAddr := "localhost:9190"                                           // gRPC endpoint
	homeDir := filepath.Join(home, "testing", "files", "simapp-validator") // Path to the keyring directory

	// Initialise codec with the necessary registerers
	interfaceRegistry, _ := cdctypes.NewInterfaceRegistryWithOptions(cdctypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
			},
			ValidatorAddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
			},
		},
	})
	std.RegisterInterfaces(interfaceRegistry)
	authtypes.RegisterInterfaces(interfaceRegistry)
	clienttypes.RegisterInterfaces(interfaceRegistry)
	groth16.RegisterInterfaces(interfaceRegistry)
	channeltypesv2.RegisterInterfaces(interfaceRegistry)
	appCodec := codec.NewProtoCodec(interfaceRegistry)

	// Keyring setup
	kr, err := keyring.New(appName, keyring.BackendTest, homeDir, nil, appCodec)
	if err != nil {
		return client.Context{}, fmt.Errorf("failed to initialize keyring: %v", err)
	}

	rec, err := kr.List()
	if err != nil {
		fmt.Println(err, "keyring list error")
	}
	addr, err := rec[0].GetAddress()
	if err != nil {
		fmt.Println(err, "keyring address error")
	}

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return client.Context{}, fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	if err != nil {
		return client.Context{}, fmt.Errorf("failed to create tendermint client: %v", err)
	}

	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           append(authtx.DefaultSignModes, legacysigning.SignMode_SIGN_MODE_TEXTUAL),
		TextualCoinMetadataQueryFn: txmodule.NewGRPCCoinMetadataQueryFn(conn),
	}

	txConfig, err := authtx.NewTxConfigWithOptions(appCodec, txConfigOpts)
	if err != nil {
		return client.Context{}, fmt.Errorf("failed to create tx config: %v", err)
	}

	cometNode, err := client.NewClientFromNode(cometNodeURI)
	if err != nil {
		return client.Context{}, err
	}

	// Initialize ClientContext
	clientCtx := client.Context{}.
		WithChainID(chainID).
		WithKeyring(kr).
		WithHomeDir(homeDir).
		WithGRPCClient(conn).
		WithFromAddress(addr).
		WithFromName(rec[0].Name).
		WithSkipConfirmation(true).
		WithInterfaceRegistry(interfaceRegistry).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithKeyring(kr).
		WithTxConfig(txConfig).
		WithBroadcastMode("sync").
		WithClient(cometNode).
		WithCodec(appCodec)

	return clientCtx, nil
}

// GetFactory returns an instance of tx.Factory that is configured with this Broadcaster's CosmosChain
// and the provided user. ConfigureFactoryOptions can be used to specify arbitrary options to configure the returned
// factory.
func GetFactory(clientContext client.Context, user string, factoryOptions tx.Factory) (tx.Factory, error) {
	sdkAdd, err := sdk.AccAddressFromBech32(user)
	if err != nil {
		return tx.Factory{}, err
	}

	account, err := clientContext.AccountRetriever.GetAccount(clientContext, sdkAdd)
	if err != nil {
		return tx.Factory{}, err
	}

	return defaultTxFactory(clientContext, account), nil
}

// defaultTxFactory creates a new Factory with default configuration.
func defaultTxFactory(clientCtx client.Context, account client.Account) tx.Factory {
	return tx.Factory{}.
		WithAccountNumber(account.GetAccountNumber()).
		WithSequence(account.GetSequence()).
		WithSignMode(legacysigning.SignMode_SIGN_MODE_DIRECT).
		WithGas(flags.DefaultGasLimit).
		WithGasPrices("0.0001stake").
		WithMemo("interchaintest").
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithKeybase(clientCtx.Keyring).
		WithChainID(clientCtx.ChainID).
		WithSimulateAndExecute(false)
}

// BroadcastMessages broadcasts the provided messages to the given chain and signs them on behalf of the provided user.
// Once the broadcast response is returned, we wait for two blocks to be created on chain.
func BroadcastMessages(clientContext client.Context, user string, gas uint64, msgs ...interface {
	ProtoMessage()
	Reset()
	String() string
}) (*sdk.TxResponse, error) {
	// Create a tx.Factory instance and apply the factoryOptions function
	factory := tx.Factory{}
	factoryOptions := factory.WithGas(gas)

	f, err := GetFactory(clientContext, user, factoryOptions)
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}
	clientContext.Output = buffer
	clientContext.WithOutput(buffer)

	if err := tx.BroadcastTx(clientContext, f, msgs...); err != nil {
		return &sdk.TxResponse{}, err
	}

	if buffer.Len() == 0 {
		return nil, fmt.Errorf("empty buffer, transaction has not been executed yet")
	}
	// unmarshall buffer into txresponse
	var txResp sdk.TxResponse
	if err := clientContext.Codec.UnmarshalJSON(buffer.Bytes(), &txResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tx response: %v", err)
	}
	return getFullyPopulatedResponse(clientContext, txResp.TxHash)
}

type User interface {
	KeyName() string
	FormattedAddress() string
}

// getFullyPopulatedResponse returns a fully populated sdk.TxResponse.
// the QueryTx function is periodically called until a tx with the given hash
// has been included in a block.
func getFullyPopulatedResponse(cc client.Context, txHash string) (*sdk.TxResponse, error) {
	var resp sdk.TxResponse
	err := WaitForCondition(time.Second*60, time.Second*5, func() (bool, error) {
		fmt.Printf("querying for txHash: %v\n", txHash)
		fullyPopulatedTxResp, err := authtx.QueryTx(cc, txHash)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Printf("tx not found yet so retrying...\n")
				return false, nil
			}
			return false, err
		}

		resp = *fullyPopulatedTxResp
		return true, nil
	})
	return &resp, err
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
