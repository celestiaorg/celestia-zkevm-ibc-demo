package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	abci "github.com/cometbft/cometbft/abci/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getTransactOpts(key *ecdsa.PrivateKey, chainID *big.Int, ethClient *ethclient.Client) *bind.TransactOpts {
	fromAddress := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		panic(err)
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		panic(err)
	}

	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.GasPrice = gasPrice

	return txOpts
}

func getTxReceipt(ctx context.Context, ethClient *ethclient.Client, hash ethcommon.Hash) (*ethtypes.Receipt, error) {
	var receipt *ethtypes.Receipt
	var err error
	err = utils.WaitForCondition(time.Second*30, time.Second, func() (bool, error) {
		receipt, err = ethClient.TransactionReceipt(ctx, hash)
		if err != nil {
			return false, nil
		}
		return receipt != nil, nil
	})

	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// getEvmEvent parses the logs in the given receipt and returns the first event
// that can be parsed.
func getEvmEvent[T any](receipt *ethtypes.Receipt, parseFn func(log ethtypes.Log) (*T, error)) (event *T, err error) {
	for _, l := range receipt.Logs {
		event, err = parseFn(*l)
		if err == nil && event != nil {
			break
		}
	}

	if event == nil {
		err = fmt.Errorf("event not found")
	}

	return
}

// getAttributeByKey returns the first event attribute with the given key.
func getAttributeByKey(attributes []abci.EventAttribute, key string) (ea abci.EventAttribute, isFound bool) {
	idx := slices.IndexFunc(attributes, func(a abci.EventAttribute) bool { return a.Key == key })
	if idx == -1 {
		return abci.EventAttribute{}, false
	}
	return attributes[idx], true
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
