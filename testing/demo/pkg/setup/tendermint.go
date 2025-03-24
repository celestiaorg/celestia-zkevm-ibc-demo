package main

import (
	"context"
	"fmt"

	"os"
	"os/exec"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CreateTendermintLightClient creates the Tendermint light client on the EVM roll-up.
func CreateTendermintLightClient() error {
	err := utils.CheckSimappNodeHealth(simappRPC, 10)
	if err != nil {
		return fmt.Errorf("simapp node is not healthy, please ensure it is running correctly: %w", err)
	}

	err = utils.CheckEthereumNodeHealth(ethereumRPC)
	if err != nil {
		return fmt.Errorf("ethereum node is not healthy, please ensure it is running correctly: %w", err)
	}

	if err := deployEurekaContracts(); err != nil {
		return err
	}

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

	if err := addClientToRouter(addresses); err != nil {
		return err
	}

	return nil
}

// deployEurekaContracts deploys all of the IBC Eureka contracts (including the
// SP1 ICS07 Tendermint light client contract) on the EVM roll-up.
func deployEurekaContracts() error {
	cmd := exec.Command("forge", "script", "E2ETestDeploy.s.sol:E2ETestDeploy", "--rpc-url", "http://localhost:8545", "--private-key", "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a", "--broadcast")
	cmd.Env = append(cmd.Env, "PRIVATE_KEY=0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a")
	cmd.Dir = "./solidity-ibc-eureka/scripts"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Deploying IBC Eureka smart contracts on the EVM roll-up...\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy contracts: %v", err)
	}
	fmt.Printf("Deployed IBC Eureka smart contracts on the EVM roll-up.\n")

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Contract Addresses: \n%v\n", addresses)

	return nil
}

// addClientToRouter adds the Tendermint light client to the router contract on
// the EVM roll-up.
//
// Note this also registers the counterparty.
func addClientToRouter(addresses utils.ContractAddresses) error {
	key, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to convert private key: %v", err)
	}

	counterpartyInfo := ics26router.IICS02ClientMsgsCounterpartyInfo{
		ClientId:     groth16ClientID,
		MerklePrefix: merklePrefix,
	}
	tmLightClientAddress := ethcommon.HexToAddress(addresses.ICS07Tendermint)

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum client: %v", err)
	}

	router, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return fmt.Errorf("failed to instantiate ICS router contract: %v", err)
	}

	fmt.Printf("Adding Tendermint light client to the router contract on EVM roll-up...\n")

	tx, err := router.AddClient(getTransactOpts(key, ethChainId, ethClient), tendermintClientID, counterpartyInfo, tmLightClientAddress)
	if err != nil {
		return fmt.Errorf("failed to add Tendermint light client to router: %v", err)
	}

	receipt, err := getTxReceipt(context.Background(), ethClient, tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	event, err := getEvmEvent(receipt, router.ParseICS02ClientAdded)
	if err != nil {
		return fmt.Errorf("failed to get event: %v", err)
	}

	fmt.Printf("Added Tendermint light client to the router contract on EVM roll-up with clientId %s and counterparty clientId %s\n", event.ClientId, event.CounterpartyInfo.ClientId)
	return nil
}
