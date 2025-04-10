package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cosmossdk.io/x/tx/signing"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SetupClientContext returns a Cosmos SDK client context
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

	// Check if the node is healthy
	if err := CheckSimappNodeHealth(cometNodeURI, 10); err != nil {
		return client.Context{}, fmt.Errorf("node health check failed: %w", err)
	}

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
	banktypes.RegisterInterfaces(interfaceRegistry)

	// Register IBC interfaces
	groth16.RegisterInterfaces(interfaceRegistry)
	solomachine.RegisterInterfaces(interfaceRegistry)
	ibctm.RegisterInterfaces(interfaceRegistry)
	ibctransfertypes.RegisterInterfaces(interfaceRegistry)
	ibcconnectiontypes.RegisterInterfaces(interfaceRegistry)
	ibcclienttypes.RegisterInterfaces(interfaceRegistry)
	clienttypesv2.RegisterInterfaces(interfaceRegistry)
	channeltypes.RegisterInterfaces(interfaceRegistry)
	channeltypesv2.RegisterInterfaces(interfaceRegistry)

	// Create codec
	appCodec := codec.NewProtoCodec(interfaceRegistry)

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

// defaultTxFactory returns a new tx factory with default configuration.
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

// BroadcastMessages creates a tx from the provided messages and signs them on behalf of the provided user.
func BroadcastMessages(clientContext client.Context, user string, gas uint64, msgs ...interface {
	ProtoMessage()
	Reset()
	String() string
}) (*sdk.TxResponse, error) {
	txFactory := tx.Factory{}
	factoryOptions := txFactory.WithGas(gas)
	factory, err := GetFactory(clientContext, user, factoryOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get factory: %v", err)
	}

	buffer := &bytes.Buffer{}
	clientContext.Output = buffer
	clientContext.WithOutput(buffer)

	if err := tx.BroadcastTx(clientContext, factory, msgs...); err != nil {
		return &sdk.TxResponse{}, fmt.Errorf("failed to broadcast tx: %v", err)
	}

	if buffer.Len() == 0 {
		return nil, fmt.Errorf("empty buffer, transaction has not been executed yet")
	}

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
	fmt.Printf("Waiting for transaction %s to land in a block...\n", txHash)

	err := WaitForCondition(time.Second*300, time.Second*15, func() (bool, error) {
		fullyPopulatedTxResp, err := authtx.QueryTx(cc, txHash)
		if err != nil {
			fmt.Printf("Still waiting for tx %s... (Error: %v)\n", txHash, err)
			return false, err
		}

		fmt.Printf("Transaction landed in block %d with code %d\n", fullyPopulatedTxResp.Height, fullyPopulatedTxResp.Code)
		if fullyPopulatedTxResp.Code != 0 {
			fmt.Printf("Transaction failed with code %d: %s\n", fullyPopulatedTxResp.Code, fullyPopulatedTxResp.RawLog)
		}
		resp = *fullyPopulatedTxResp
		return true, nil
	})
	return &resp, err
}

// WaitForCondition periodically executes the given function fn based on the
// provided pollingInterval. The function fn should return true if the desired
// condition is met. If the function never returns true within the timeoutAfter
// period, or fn returns an error, the condition will not have been met.
func WaitForCondition(timeoutAfter time.Duration, pollingInterval time.Duration, fn func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutAfter)
	defer cancel()

	maxRetries := 5
	backoffDuration := time.Second

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed waiting for condition after %f seconds", timeoutAfter.Seconds())
		case <-time.After(pollingInterval):
			// Add retry logic with exponential backoff for connection errors
			var lastErr error
			for retries := 0; retries < maxRetries; retries++ {
				reachedCondition, err := fn()
				if err == nil {
					if reachedCondition {
						return nil
					}
					break // No error but condition not reached, break retry loop and continue outer polling loop
				}

				lastErr = err
				// Check if error is connection-related
				if strings.Contains(err.Error(), "EOF") ||
					strings.Contains(err.Error(), "connection") ||
					strings.Contains(err.Error(), "Post") {
					fmt.Printf("Connection error detected, retrying in %v (attempt %d/%d): %v\n",
						backoffDuration, retries+1, maxRetries, err)
					time.Sleep(backoffDuration)
					backoffDuration *= 2 // Exponential backoff
					continue
				} else {
					// Not a connection error, return immediately
					return fmt.Errorf("error occurred while waiting for condition: %s", err)
				}
			}

			if lastErr != nil {
				return fmt.Errorf("error occurred while waiting for condition: %s", lastErr)
			}
		}
	}
}
