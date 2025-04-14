package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ibcerc20"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20transfer"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Global variable to store the MPT proof
var mptProof []byte
var evmTransferBlockNumber uint64

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

	err, packetCommitmentPath := sendTransferBackMsg()
	if err != nil {
		return fmt.Errorf("failed to send transfer back msg: %w", err)
	}

	commitmentsStorageKey := GetCommitmentsStorageKey(packetCommitmentPath)

	// Get the MPT proof
	proof, err := getMPTProof(commitmentsStorageKey, ethcommon.HexToAddress(addresses.ICS26Router))
	if err != nil {
		return fmt.Errorf("failed to get MPT proof: %w", err)
	}
	fmt.Printf("MPT proof: %v\n", proof)

	err = updateGroth16LightClient()
	if err != nil {
		return fmt.Errorf("failed to update Groth16 light client: %w", err)
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

func sendTransferBackMsg() (error, []byte) {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return fmt.Errorf("failed to get contract addresses: %w", err), []byte{}
	}

	ibcERC20Address, err := getIBCERC20Address()
	if err != nil {
		return fmt.Errorf("failed to get IBC ERC20 contract address: %w", err), []byte{}
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err), []byte{}
	}

	ics20Contract, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return err, []byte{}
	}

	ics26Contract, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return fmt.Errorf("failed to get ICS26Router contract address: %w", err), []byte{}
	}

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err), []byte{}
	}

	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err), []byte{}
	}

	msg := ics20transfer.IICS20TransferMsgsSendTransferMsg{
		Denom:            ibcERC20Address,
		Amount:           transferBackAmount,
		Receiver:         sender,
		TimeoutTimestamp: uint64(time.Now().Add(30 * time.Minute).Unix()),
		SourceClient:     tendermintClientID,
		Memo:             "transfer back memo",
	}
	tx, err := ics20Contract.SendTransfer(getTransactOpts(privateKey, eth), msg)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err), []byte{}
	}

	receipt, err := getTxReciept(context.Background(), eth, tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err), []byte{}
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("send transfer back msg failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs), []byte{}
	}

	event, err := GetEvmEvent(receipt, ics26Contract.ParseSendPacket)
	if err != nil {
		return fmt.Errorf("failed to get send packet event: %w", err), []byte{}
	}
	fmt.Printf("send packet event: %v\n", event)
	fmt.Print(event.Packet.Sequence, "SEQUENCE")
	fmt.Print(event.Packet.SourceClient, "SOURCE CLIENT")

	evmTransferBlockNumber = receipt.BlockNumber.Uint64()

	// concatenate sourceClient and sequence
	packetCommitmentPath := packetCommitmentPath([]byte(event.Packet.SourceClient), event.Packet.Sequence)

	fmt.Printf("packetCommitmentPath: %s\n", ethcommon.Bytes2Hex(packetCommitmentPath))

	fmt.Printf("Submit transfer back msg successfully tx hash: %s\n", tx.Hash().Hex())
	return nil, packetCommitmentPath
}

// getMPTProof queries the Reth node for a Merkle Patricia Trie proof for a given key
func getMPTProof(path ethcommon.Hash, contractAddress ethcommon.Address) ([]byte, error) {
	// Connect to the Reth node
	client, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Reth node: %w", err)
	}
	defer client.Close()

	// Step 1: keccak256(path)

	// Step 2: keccak256(pathHash ++ slot)
	storageKey := crypto.Keccak256Hash(append(path.Bytes(), []byte(ics26router.IbcStoreStorageSlot)...))
	fmt.Printf("path: %v\n", path)

	// Step 3: format args
	addressHex := contractAddress.Hex()
	keys := []string{storageKey.Hex()}
	blockHex := hexutil.EncodeUint64(evmTransferBlockNumber)

	fmt.Printf("eth_getProof args: %s, %v, %s\n", addressHex, keys, blockHex)

	// Step 4: call eth_getProof
	var result map[string]interface{}
	err = client.Client().Call(&result, "eth_getProof", contractAddress.Hex(), keys, blockHex)
	if err != nil {
		return nil, fmt.Errorf("failed to get MPT proof: %w", err)
	}
	fmt.Printf("MPT PROOF: %v\n", result)
	// The result contains the account proof and storage proofs
	// We need to extract the storage proof for our key
	if len(result) < 2 {
		return nil, fmt.Errorf("invalid proof result format")
	}

	// The storage proof is in the second element of the result
	storageProof, ok := result["result"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid storage proof format")
	}

	// Convert the proof to bytes
	proofBytes, err := json.Marshal(storageProof)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof: %w", err)
	}

	fmt.Printf("Submit transfer back msg successfully tx hash: %s\n", tx.Hash().Hex())
	// fmt.Printf("Successfully retrieved MPT proof for key: %s\n", key)
	return proofBytes, nil
}
