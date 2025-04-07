package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ibcerc20"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20transfer"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Global variable to store the MPT proof
var mptProof []byte

func main() {
	if os.Args[1] == "transfer" {
		err := transferSimAppToEVM()
		if err != nil {
			log.Fatal("Failed to transfer from SimApp to EVM roll-up: ", err)
		}
	} else if os.Args[1] == "transfer-back" {
		err := transferBack()
		if err != nil {
			log.Fatal("Failed to transfer from EVM roll-up to SimApp: ", err)
		}
	} else if os.Args[1] == "query-balance" {
		err := queryBalance()
		if err != nil {
			log.Fatal("Failed to query balance: ", err)
		}
	}
}

func transferSimAppToEVM() error {
	err := assertVerifierKeys()
	if err != nil {
		return fmt.Errorf("failed to assert verifier keys: %w", err)
	}

	msg, err := createMsgSendPacket()
	if err != nil {
		return fmt.Errorf("failed to create msg send packet: %w", err)
	}

	txHash, err := submitMsgTransfer(msg)
	if err != nil {
		return fmt.Errorf("failed to submit msg transfer: %w", err)
	}

	err = updateTendermintLightClient()
	if err != nil {
		return fmt.Errorf("failed to update Tendermint light client: %w", err)
	}

	err = relayByTx(txHash, tendermintClientID)
	if err != nil {
		return fmt.Errorf("failed to relay IBC transaction: %w", err)
	}

	err = queryBalance()
	if err != nil {
		return fmt.Errorf("failed to query balance: %w", err)
	}

	return nil
}

func transferBack() error {
	err := approveSpend()
	if err != nil {
		return fmt.Errorf("failed to approve spend: %w", err)
	}

	// TODO: we could also save the tx hash when making the transfer
	// and use it as the key for the MPT proof

	// Get the contract addresses
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return fmt.Errorf("failed to get contract addresses: %w", err)
	}

	// Get the latest transaction hash from the ICS20Transfer contract
	// This is a simplified approach - in a real implementation, you would need to
	// determine the exact transaction that represents the transfer
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}
	defer ethClient.Close()

	// Get the latest block
	latestBlock, err := ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	// Find the latest transaction to the ICS20Transfer contract
	var latestTxHash ethcommon.Hash
	for _, tx := range latestBlock.Transactions() {
		if tx.To() != nil && tx.To().Hex() == addresses.ICS20Transfer {
			latestTxHash = tx.Hash()
			break
		}
	}

	if latestTxHash == (ethcommon.Hash{}) {
		return fmt.Errorf("no transaction found to ICS20Transfer contract")
	}

	// Get the key for the MPT proof
	key, err := getTransferKeyFromTxHash(latestTxHash)
	if err != nil {
		return fmt.Errorf("failed to get transfer key: %w", err)
	}

	// Attach the MPT proof to the transfer packet
	err = attachMPTProofToTransfer(key, addresses.ICS20Transfer)
	if err != nil {
		return fmt.Errorf("failed to attach MPT proof: %w", err)
	}

	err = sendTransferBackMsg()
	if err != nil {
		return fmt.Errorf("failed to send transfer back msg: %w", err)
	}

	return nil
}

func approveSpend() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ibcERC20Address, err := getIBCERC20Address()
	if err != nil {
		return fmt.Errorf("failed to get IBC ERC20 contract address: %w", err)
	}

	erc20, err := ibcerc20.NewContract(ibcERC20Address, ethClient)
	if err != nil {
		return err
	}

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}

	tx, err := erc20.Approve(getTransactOpts(privateKey, eth), ethcommon.HexToAddress(addresses.ICS20Transfer), transferBackAmount)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	receipt, err := getTxReciept(context.Background(), eth, tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("approve failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs)
	}

	allowance, err := erc20.Allowance(getCallOpts(), ethcommon.HexToAddress(receiver), ethcommon.HexToAddress(addresses.ICS20Transfer))
	if err != nil {
		return fmt.Errorf("failed to get allowance: %w", err)
	}

	if allowance.Cmp(transferBackAmount) != 0 {
		return fmt.Errorf("allowance is not correct: %v", allowance)
	} else {
		fmt.Printf("Allowed %v tokens to be spent by the ICS20Transfer contract\n", allowance)
	}

	return nil
}

func sendTransferBackMsg() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

	ibcERC20Address, err := getIBCERC20Address()
	if err != nil {
		return fmt.Errorf("failed to get IBC ERC20 contract address: %w", err)
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ics20Contract, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return err
	}

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}

	// Create a memo that includes the MPT proof
	memo := "transfer back memo"
	if len(mptProof) > 0 {
		// Encode the proof as base64 and include it in the memo
		proofBase64 := base64.StdEncoding.EncodeToString(mptProof)
		memo = fmt.Sprintf("%s|proof:%s", memo, proofBase64)
		fmt.Printf("Included MPT proof in memo (length: %d)\n", len(proofBase64))
	}

	msg := ics20transfer.IICS20TransferMsgsSendTransferMsg{
		Denom:            ibcERC20Address,
		Amount:           transferBackAmount,
		Receiver:         sender,
		TimeoutTimestamp: uint64(time.Now().Add(30 * time.Minute).Unix()),
		SourceClient:     tendermintClientID,
		Memo:             memo,
	}
	tx, err := ics20Contract.SendTransfer(getTransactOpts(privateKey, eth), msg)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	receipt, err := getTxReciept(context.Background(), eth, tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("send transfer back msg failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs)
	}

	fmt.Printf("send transfer back msg success tx hash: %s\n", tx.Hash().Hex())
	return nil
}

// getMPTProof queries the Reth node for a Merkle Patricia Trie proof for a given key
func getMPTProof(key []byte, contractAddress string) ([]byte, error) {
	// Connect to the Reth node
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Reth node: %w", err)
	}
	defer client.Close()

	// Convert key to hex string
	keyHex := ethcommon.Bytes2Hex(key)

	// Call the getProof JSON RPC method
	var result []interface{}
	err = client.Client().Call(&result, "eth_getProof",
		contractAddress,  // contract address
		[]string{keyHex}, // keys to get proof for
		"latest")         // block number or "latest"

	if err != nil {
		return nil, fmt.Errorf("failed to get MPT proof: %w", err)
	}

	// The result contains the account proof and storage proofs
	// We need to extract the storage proof for our key
	if len(result) < 2 {
		return nil, fmt.Errorf("invalid proof result format")
	}

	// The storage proof is in the second element of the result
	storageProof, ok := result[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid storage proof format")
	}

	// Convert the proof to bytes
	proofBytes, err := json.Marshal(storageProof)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof: %w", err)
	}

	fmt.Printf("Successfully retrieved MPT proof for key: %s\n", keyHex)
	return proofBytes, nil
}

// attachMPTProofToTransfer attaches the MPT proof to the transfer packet
func attachMPTProofToTransfer(key []byte, contractAddress string) error {
	// Get the MPT proof
	proof, err := getMPTProof(key, contractAddress)
	if err != nil {
		return fmt.Errorf("failed to get MPT proof: %w", err)
	}

	// Store the proof for later use
	mptProof = proof
	fmt.Printf("MPT proof length: %d bytes\n", len(proof))

	return nil
}

// getTransferKeyFromTxHash extracts the key for the MPT proof from a transaction hash
func getTransferKeyFromTxHash(txHash ethcommon.Hash) ([]byte, error) {
	// Connect to the Reth node
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Reth node: %w", err)
	}
	defer client.Close()

	// Get the transaction receipt
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// The key for the MPT proof is typically the hash of the transaction
	// This is a simplified approach - in a real implementation, you would need to
	// determine the exact key based on the contract's storage layout
	key := receipt.TxHash.Bytes()

	fmt.Printf("Extracted key from transaction: %s\n", ethcommon.Bytes2Hex(key))
	return key, nil
}
