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
		err := assertBalances()
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

	err = updateBalances()
	if err != nil {
		return fmt.Errorf("failed to update balances: %w", err)
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

	err = assertBalances()
	if err != nil {
		return fmt.Errorf("failed to query balance: %w", err)
	}

	return nil
}

func transferBack() error {
	// publicValuesPassed := []byte{130, 145, 146, 189, 207, 51, 223, 212, 130, 247, 223, 62, 65, 247, 138, 53, 31, 95, 170, 173, 170, 236, 62, 159, 130, 119, 143, 80, 106, 167, 179, 22, 192, 160, 132, 95, 74, 173, 130, 202, 126, 212, 70, 86, 85, 143, 210, 219, 89, 49, 82, 157, 181, 31, 251, 70, 16, 142, 6, 45, 220, 33, 10, 160, 11, 0, 0, 0, 0, 0, 0, 0, 95, 138, 224, 134, 39, 215, 175, 19, 32, 250, 40, 109, 36, 247, 105, 227, 161, 116, 139, 86, 93, 16, 150, 118, 125, 134, 74, 189, 151, 160, 245, 225, 198, 93, 32, 141, 113, 80, 76, 21, 135, 241, 141, 169, 228, 162, 152, 254, 108, 50, 154, 206, 142, 171, 26, 20, 246, 72, 206, 18, 231, 241, 210, 7, 160, 249, 255, 241, 214, 124, 123, 20, 64, 160, 98, 226, 38, 85, 182, 33, 72, 223, 192, 179, 235, 220, 174, 3, 82, 241, 225, 149, 171, 239, 198, 123, 203, 191, 95, 211, 74, 111, 114, 241, 238, 86, 41, 110, 6, 65, 89, 19, 0, 61, 113, 161, 156, 146, 161, 255, 241, 93, 24, 53, 61, 132, 219, 252, 51, 187, 189, 127, 91, 94, 209, 149, 218, 80, 11, 137, 171, 128, 52, 55, 18, 182, 95, 184, 254, 61, 50, 170, 65, 107, 231, 72, 176, 133, 230, 102, 127, 146, 195, 230, 17, 220, 173, 252, 239, 118, 2, 127, 3, 73, 125, 68, 22, 93, 41, 19, 21, 156, 178, 23, 132, 42, 54, 97, 204, 90, 124, 42, 64, 239, 203, 114, 217, 229, 27, 20, 155, 197, 96, 10, 40, 179, 109, 58, 252, 176, 7, 68, 54, 42, 19, 168, 174, 57, 167, 202, 149, 188, 202, 53, 191, 203, 115, 196, 186, 43, 127, 51, 72, 228, 93, 213, 198, 150, 234, 233, 45, 55, 1, 216, 72, 83, 251, 8, 66, 5, 8, 65, 169, 178, 52, 26, 120, 138, 228, 142, 116, 74, 7, 247, 116, 52, 82, 189, 193, 78, 4, 49, 107, 62, 211, 62, 27, 103, 102, 2, 204, 163, 111, 242, 107, 241, 72, 75, 2, 239, 63, 95, 199, 91, 121, 42, 80, 6, 140, 92, 133, 174, 244, 235, 17, 107, 82, 66, 136, 220, 208, 161, 78, 191, 162, 239, 250, 127, 41, 68, 8, 71, 227, 148, 169, 10, 110, 208, 158, 243, 139, 158, 122, 80, 240, 44, 87, 7, 227, 251, 121, 148, 123, 94, 206, 229, 239, 72, 209, 51, 33, 39, 70, 228, 88, 200, 88, 88, 169, 112, 127, 189, 121, 209, 101, 253, 163, 171, 148, 78, 60, 154, 46, 62, 208, 88, 95, 66, 108, 21, 181, 82, 58, 207, 15, 0, 0, 0, 0, 0, 0, 0}

	// publicValues, err := DecodePublicValues(publicValuesPassed)
	// if err != nil {
	// 	return fmt.Errorf("failed to decode public values: %w", err)
	// }

	// Generate vkeyHash
	// vkeyHasher := sha256.New()
	// vkeyHasher.Write(vkeyBytes)
	// vkeyHash := "0x" + hex.EncodeToString(vkeyHasher.Sum(nil))

	// Generate committedValuesDigest
	// valuesHasher := sha256.New()
	// valuesHasher.Write(publicValuesPassed)

	// The Plonk and Groth16 verifiers operate over a 254 bit field, so we need to zero
	// out the first 3 bits. The same logic happens in the SP1 Ethereum verifier contract.
	// publicValuesPassed[0] &= 0x1F

	// committedValuesDigest := "0x" + hex.EncodeToString(publicValuesPassed)
	// fmt.Println(committedValuesDigest, "COMMITTED VALUES DIGEST")

	// call evm prover

	// proof := gnark.NewProof(ecc.BN254)
	// _, err = proof.ReadFrom(bytes.NewReader(header.StateTransitionProof))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to read proof: %w", err)
	// }

	// fmt.Printf("Verifying state transition proof...\n")
	// err = gnark.Verify(proof, vk, witness)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to verify proof: %w", err)
	// }

	err := approveSpend()
	if err != nil {
		return fmt.Errorf("failed to approve spend: %w", err)
	}

	err = updateBalances()
	if err != nil {
		return fmt.Errorf("failed to update balances: %w", err)
	}

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return fmt.Errorf("failed to get contract addresses: %w", err)
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}
	defer ethClient.Close()

	sendPacketEvent, evmTransferBlockNumber, err := sendTransferBackMsg()
	if err != nil {
		return fmt.Errorf("failed to send transfer back msg: %w", err)
	}

	// Generate the path for the packet commitment which is required for the commitment proof generation.
	packetCommitmentPath := packetCommitmentPath(sendPacketEvent.Packet.SourceClient, sendPacketEvent.Packet.Sequence)

	proof, err := getMPTProof(packetCommitmentPath, addresses.ICS26Router, evmTransferBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to get MPT proof: %w", err)
	}

	err = updateGroth16LightClient(evmTransferBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to update Groth16 light client: %w", err)
	}

	err = relayFromEvmToSimapp(sendPacketEvent, proof, evmTransferBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to relay from EVM to SimApp: %w", err)
	}

	err = assertBalances()
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

func sendTransferBackMsg() (*ics26router.ContractSendPacket, uint64, error) {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get contract addresses: %w", err)
	}

	ibcERC20Address, err := getIBCERC20Address()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get IBC ERC20 contract address: %w", err)
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ics20Contract, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get ICS20Transfer contract address: %w", err)
	}

	ics26Contract, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get ICS26Router contract address: %w", err)
	}

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(receiverPrivateKey))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse private key: %w", err)
	}

	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, privateKey)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create Ethereum client: %w", err)
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
		return nil, 0, fmt.Errorf("failed to create transaction: %w", err)
	}

	receipt, err := getTxReciept(context.Background(), eth, tx.Hash())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return nil, 0, fmt.Errorf("send transfer back msg failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs)
	}

	// Parse the send packet event from the receipt
	sendPacketEvent, err := GetEvmEvent(receipt, ics26Contract.ParseSendPacket)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get send packet event: %w", err)
	}

	fmt.Printf("Submit transfer back msg successfully tx hash: %s\n", tx.Hash().Hex())
	return sendPacketEvent, receipt.BlockNumber.Uint64(), nil
}
