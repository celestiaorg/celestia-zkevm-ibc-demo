package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20transfer"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getTransactOpts(key *ecdsa.PrivateKey, chain ethereum.Ethereum) *bind.TransactOpts {
	ethClient, err := ethclient.Dial(chain.RPC)
	if err != nil {
		log.Fatal(err)
	}

	fromAddress := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		nonce = 0
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(key, chain.ChainID)
	if err != nil {
		log.Fatal(err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.GasPrice = gasPrice
	txOpts.GasLimit = 5_000_000

	return txOpts
}

func getTxReciept(ctx context.Context, chain ethereum.Ethereum, hash ethcommon.Hash) (*ethtypes.Receipt, error) {
	ethClient, err := ethclient.Dial(chain.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	var receipt *ethtypes.Receipt
	err = utils.WaitForCondition(time.Second*30, time.Second, func() (bool, error) {
		receipt, err = ethClient.TransactionReceipt(ctx, hash)
		if err != nil {
			return false, nil
		}

		return receipt != nil, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch receipt: %v", err)
	}

	return receipt, nil
}

// getIBCERC20Address returns the address of the IBC ERC20 contract on the Ethereum chain.
// This is the ERC20 contract that has the tokens transfered from Celestia to Ethereum.
func getIBCERC20Address() (ethcommon.Address, error) {
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to connect to Ethereum: %w", err)
	}
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return ethcommon.Address{}, err
	}

	ics20Transfer, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to create ICS20Transfer contract: %w", err)
	}

	denomOnEthereum := transfertypes.NewDenom(denom, transfertypes.NewHop(transfertypes.PortID, tendermintClientID))

	ibcERC20Address, err := ics20Transfer.IbcERC20Contract(nil, denomOnEthereum.Path())
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to get IBC ERC20 contract address: %w", err)
	}
	return ibcERC20Address, nil
}

// From https://medium.com/@zhuytt4/verify-the-owner-of-safe-wallet-with-eth-getproof-7edc450504ff
func GetCommitmentsStorageKey(path []byte) ethcommon.Hash {
	commitmentStorageSlot := ethcommon.FromHex(ics26router.IbcStoreStorageSlot)

	pathHash := crypto.Keccak256(path)

	// zero pad both to 32 bytes
	paddedSlot := ethcommon.LeftPadBytes(commitmentStorageSlot, 32)

	// keccak256(h(k) . slot)
	return crypto.Keccak256Hash(pathHash, paddedSlot)
}

func packetCommitmentPath(clientId []byte, sequence uint64) []byte {
	// Convert sequence to big endian bytes (8 bytes)
	sequenceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sequenceBytes, sequence)

	// The path is: clientId + uint8(1) + sequenceBytes
	path := append(clientId, byte(1))
	path = append(path, sequenceBytes...)

	return path
}

// GetEvmEvent parses the logs in the given receipt and returns the first event that can be parsed
func GetEvmEvent[T any](receipt *ethtypes.Receipt, parseFn func(log ethtypes.Log) (*T, error)) (event *T, err error) {
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
