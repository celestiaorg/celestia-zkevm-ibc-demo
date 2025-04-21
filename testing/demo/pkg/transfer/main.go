package main

import (
	"context"
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
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Global variable to store the MPT proof
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

	err, packetCommitmentPath, sendPacketEvent := sendTransferBackMsg()
	if err != nil {
		return fmt.Errorf("failed to send transfer back msg: %w", err)
	}

	// Get the MPT proof
	proof, err := getMPTProof(packetCommitmentPath, addresses.ICS26Router)
	if err != nil {
		return fmt.Errorf("failed to get MPT proof: %w", err)
	}

	err = updateGroth16LightClient(evmTransferBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to update Groth16 light client: %w", err)
	}

	// everything in proof should be 32 byte value (uint256 or bytes32)
	err = relayFromEvmToSimapp(sendPacketEvent, proof, evmTransferBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to relay from EVM to SimApp: %w", err)
	}

	err = queryBalance()
	if err != nil {
		return fmt.Errorf("failed to query balance: %w", err)
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

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(receiverPrivateKey))
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

func sendTransferBackMsg() (error, []byte, *ics26router.ContractSendPacket) {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return fmt.Errorf("failed to get contract addresses: %w", err), []byte{}, nil
	}

	ibcERC20Address, err := getIBCERC20Address()
	if err != nil {
		return fmt.Errorf("failed to get IBC ERC20 contract address: %w", err), []byte{}, nil
	}
	fmt.Println("ibcERC20Address: ", ibcERC20Address)

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err), []byte{}, nil
	}

	ics20Contract, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return err, []byte{}, nil
	}

	ics26Contract, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return fmt.Errorf("failed to get ICS26Router contract address: %w", err), []byte{}, nil
	}

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(receiverPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err), []byte{}, nil
	}

	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err), []byte{}, nil
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
		return fmt.Errorf("failed to create transaction: %w", err), []byte{}, nil
	}

	receipt, err := getTxReciept(context.Background(), eth, tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err), []byte{}, nil
	}
	evmTransferBlockNumber = receipt.BlockNumber.Uint64()

	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("send transfer back msg failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs), []byte{}, nil
	}

	// Parse the send packet event from the receipt
	sendPacketEvent, err := GetEvmEvent(receipt, ics26Contract.ParseSendPacket)
	if err != nil {
		return fmt.Errorf("failed to get send packet event: %w", err), []byte{}, nil
	}

	// Generate the path for the packet commitment which is required for the commitment proof generation
	packetCommitmentPath := packetCommitmentPath(sendPacketEvent.Packet.SourceClient, sendPacketEvent.Packet.Sequence)

	fmt.Printf("Submit transfer back msg successfully tx hash: %s\n", tx.Hash().Hex())
	return nil, packetCommitmentPath, sendPacketEvent
}
