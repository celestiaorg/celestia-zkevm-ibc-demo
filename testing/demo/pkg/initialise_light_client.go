package pkg

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacysigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
)

func InitializeLightClient() error {
	fmt.Println("Initializing the groth16 light client")
	// ETH SET UP
	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum client: %v", err)
	}

	// retrieve the genesis block
	genesisBlock, err := ethClient.BlockByNumber(context.Background(), big.NewInt(0))
	if err != nil {
		return fmt.Errorf("failed to get genesis block: %v", err)
	}

	latestBlock, err := ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %v", err)
	}

	clientState := groth16.NewClientState(latestBlock.Number().Uint64(), []byte{}, []byte{}, []byte{}, genesisBlock.Root().Bytes())
	clientStateAny, err := cdctypes.NewAnyWithValue(clientState)
	if err != nil {
		return fmt.Errorf("failed to create client state any: %v", err)
	}
	latestBlockTime := time.Unix(int64(latestBlock.Time()), 0)
	consensusState := groth16.NewConsensusState(latestBlockTime, latestBlock.Root().Bytes())
	consensusStateAny, err := cdctypes.NewAnyWithValue(consensusState)
	if err != nil {
		return fmt.Errorf("failed to create consensus state any: %v", err)
	}

	// relayer address registered in simapp
	relayer := "cosmos16p2hfunj38hucvlzx4fp4dj42y6gddcxj60yxn"

	msg := &clienttypes.MsgCreateClient{
		ClientState:    clientStateAny,
		ConsensusState: consensusStateAny,
		Signer:         relayer,
	}

	// simapp address
	clientCtx, err := SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	_, err = BroadcastMessages(clientCtx, relayer, 200, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast message: %v", err)
	}
	return nil
}

// SetupClientContext initializes a Cosmos SDK ClientContext
func SetupClientContext() (client.Context, error) {
	// Chain-specific configurations
	chainID := "zkibc-demo"
	nodeURI := "http://localhost:26657"       // RPC endpoint
	homeDir := "../../files/simapp-validator" // Path to Cosmos configuration directory
	appName := "celestia-zkevm-ibc-demo"      // Name of the application from the genesis file

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
	appCodec := codec.NewProtoCodec(interfaceRegistry)

	// Keyring setup
	kr, err := keyring.New(appName, keyring.BackendTest, homeDir, nil, appCodec)
	if err != nil {
		return client.Context{}, fmt.Errorf("failed to initialize keyring: %v", err)
	}

	// Initialize ClientContext
	clientCtx := client.Context{}.
		WithChainID(chainID).
		WithKeyring(kr).
		WithNodeURI(nodeURI).
		WithHomeDir(homeDir)

	// Create a Tendermint RPC client and set it in ClientContext
	rpcClient, err := client.NewClientFromNode(nodeURI)
	if err != nil {
		return client.Context{}, fmt.Errorf("failed to create RPC client: %v", err)
	}
	clientCtx = clientCtx.WithClient(rpcClient)

	// Set codec (you need to pass your own codec)
	clientCtx = clientCtx.WithCodec(appCodec)

	// Set input/output for CLI commands (if applicable)
	clientCtx = clientCtx.WithInput(nil).WithOutput(nil)

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
		WithGasPrices("0.002").
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
	// sdk.GetConfig().SetBech32PrefixForAccount(chain.Config().Bech32Prefix, chain.Config().Bech32Prefix+sdk.PrefixPublic)
	// sdk.GetConfig().SetBech32PrefixForValidator(
	// 	chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator,
	// 	chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic,
	// )

	// Create a tx.Factory instance and apply the factoryOptions function
	factory := tx.Factory{}
	factoryOptions := factory.WithGas(gas)

	f, err := GetFactory(clientContext, user, factoryOptions)
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}
	clientContext.Output = buffer

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

	return &txResp, nil
}

type User interface {
	KeyName() string
	FormattedAddress() string
}
