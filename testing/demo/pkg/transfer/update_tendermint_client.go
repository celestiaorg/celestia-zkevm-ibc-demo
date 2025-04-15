package main

import (
	"context"
	"fmt"
	"strings"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// updateTendermintLightClient submits a MsgUpdateClient to the Tendermint light
// client on the EVM roll-up.
func updateTendermintLightClient() error {
	fmt.Printf("Updating Tendermint light client on EVM roll-up...\n")

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	icsRouter, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return err
	}
	faucet, err := crypto.ToECDSA(ethcommon.FromHex(receiverPrivateKey))
	if err != nil {
		return err
	}
	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, faucet)
	if err != nil {
		return err
	}
	updateMsg, err := getUpdateMsg()
	if err != nil {
		return fmt.Errorf("failed to get update msg: %w", err)
	}

	fmt.Printf("Submitting UpdateClient on EVM roll-up...\n")
	ethTx, err := icsRouter.UpdateClient(getTransactOpts(faucet, eth), tendermintClientID, updateMsg)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	receipt, err := getTxReciept(context.Background(), eth, ethTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("UpdateClient tx failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs)
	}
	fmt.Printf("Updated Tendermint light client in block %v.\n", receipt.BlockNumber.Uint64())
	return nil
}

func getUpdateMsg() (updateMsg []byte, err error) {
	arguments, err := getUpdateClientArguments()
	if err != nil {
		return nil, err
	}
	verifierKey, err := getProverSTFKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get prover state transition verifier key: %w", err)
	}
	resp, err := getProofResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to get proof response: %w", err)
	}

	updateMsg, err = arguments.Pack(struct {
		Sp1Proof sp1proof
	}{
		Sp1Proof: sp1proof{
			VKey:         verifierKey,
			PublicValues: resp.PublicValues,
			Proof:        resp.Proof,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error packing msg %w", err)
	}
	return updateMsg, nil
}

func getUpdateClientArguments() (abi.Arguments, error) {
	var updateClientABI = "[{\"type\":\"function\",\"name\":\"updateClient\",\"stateMutability\":\"pure\",\"inputs\":[{\"name\":\"o3\",\"type\":\"tuple\",\"internalType\":\"struct IUpdateClientMsgs.MsgUpdateClient\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"struct ISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]}],\"outputs\":[]}]"

	parsed, err := abi.JSON(strings.NewReader(updateClientABI))
	if err != nil {
		return nil, err
	}

	return parsed.Methods["updateClient"].Inputs, nil
}

func getProofResponse() (resp *proverclient.ProveStateTransitionResponse, err error) {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return nil, err
	}

	celestiaProverConn, err := grpc.NewClient(celestiaProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer celestiaProverConn.Close()
	proverClient := proverclient.NewProverClient(celestiaProverConn)

	fmt.Printf("Requesting celestia-prover state transition proof...\n")

	resp, err = proverClient.ProveStateTransition(context.Background(), &proverclient.ProveStateTransitionRequest{ClientId: addresses.ICS07Tendermint})
	if err != nil {
		return nil, fmt.Errorf("failed to get state transition proof: %w", err)
	}
	fmt.Printf("Received celestia-prover state transition proof.\n")
	return resp, nil
}

// sp1proof represents the proof structure used in the update client message
type sp1proof struct {
	VKey         [32]byte
	PublicValues []byte
	Proof        []byte
}
