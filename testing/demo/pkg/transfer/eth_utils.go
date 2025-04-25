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
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// StorageProof contains MPT proof for a specific storage slot.
// It verifies the existence and value of a storage key in the EVM state.
type StorageProof struct {
	// The key of the storage
	Key common.Hash `json:"key"`
	// The value of the storage
	Value hexutil.Big `json:"value"`
	// The proof of the storage
	Proof []hexutil.Bytes `json:"proof"`
}

// EthGetProofResponse is the response from the eth_getProof RPC call.
type EthGetProofResponse struct {
	AccountProof []hexutil.Bytes `json:"accountProof"`
	Address      common.Address  `json:"address"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`

	StorageHash  common.Hash    `json:"storageHash"`
	StorageProof []StorageProof `json:"storageProof"`
}

// MptProof is the proof of the commitment of the packet on the EVM chain.
type MptProof struct {
	AccountProof []hexutil.Bytes `json:"accountProof"`
	Address      common.Address  `json:"address"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []hexutil.Bytes `json:"storageProof"`
	StorageKey   common.Hash     `json:"storageKey"`
	StorageValue hexutil.Big     `json:"storageValue"`
}

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

	// zero pad to 32 bytes
	paddedSlot := ethcommon.LeftPadBytes(commitmentStorageSlot, 32)

	// keccak256(h(k) . slot)
	return crypto.Keccak256Hash(pathHash, paddedSlot)
}

// packetCommitmentPath returns the path of the packet commitment on the EVM chain.
func packetCommitmentPath(clientId string, sequence uint64) []byte {
	// Convert sequence to big endian bytes (8 bytes)
	sequenceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sequenceBytes, sequence)

	// The path is: clientId bytes + uint8(1) + sequenceBytes
	path := make([]byte, 0)
	path = append(path, []byte(clientId)...) // Convert string to bytes first
	path = append(path, byte(1))             // Marker byte for packet commitment
	path = append(path, sequenceBytes...)    // Sequence in big endian

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
		return nil, fmt.Errorf("event not found")
	}

	return event, nil
}

// getMPTProof queries the Reth node for a Merkle Patricia Trie proof for a given key
func getMPTProof(packetCommitmentPath []byte, contractAddress string, evmTransferBlockNumber uint64) (MptProof, error) {
	commitmentsStorageKey := GetCommitmentsStorageKey(packetCommitmentPath)

	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return MptProof{}, fmt.Errorf("failed to connect to Reth node: %w", err)
	}
	defer client.Close()

	// Generate the proof for the given path
	var result EthGetProofResponse
	err = client.Client().Call(&result, "eth_getProof", contractAddress, []string{commitmentsStorageKey.Hex()}, hexutil.EncodeUint64(evmTransferBlockNumber))
	if err != nil {
		return MptProof{}, fmt.Errorf("failed to get MPT proof: %w", err)
	}

	// Find the proof for our specific storage key
	var targetProof StorageProof
	for _, proof := range result.StorageProof {
		if proof.Key == commitmentsStorageKey {
			targetProof = proof
			break
		} else {
			return MptProof{}, fmt.Errorf("proof key does not match the path: %x", proof.Key)
		}
	}

	proof := MptProof{
		AccountProof: result.AccountProof,
		Address:      result.Address,
		Balance:      result.Balance,
		CodeHash:     result.CodeHash,
		Nonce:        result.Nonce,
		StorageHash:  result.StorageHash,
		StorageProof: targetProof.Proof,
		StorageKey:   targetProof.Key,
		StorageValue: targetProof.Value,
	}

	return proof, nil
}
