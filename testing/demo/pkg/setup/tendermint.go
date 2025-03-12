package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"math/big"
	"os"
	"os/exec"
	"time"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
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
	fmt.Println("Deployed IBC Eureka smart contracts on the EVM roll-up.\n")

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Contract Addresses: \n%v\n", addresses)

	return nil
}

// addClientToRouter adds the Tendermint light client to the router contract on
// the EVM roll-up.
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

	receipt := getTxReceipt(context.Background(), ethClient, tx.Hash())
	event, err := getEvmEvent(receipt, router.ParseICS02ClientAdded)
	if err != nil {
		return fmt.Errorf("failed to get event: %v", err)
	}
	fmt.Printf("Added Tendermint lightclient to the router contract on EVM roll-up with clientId %s and counterparty clientId %s\n", event.ClientId, event.CounterpartyInfo.ClientId)
	return nil
}

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

func getTxReceipt(ctx context.Context, ethClient *ethclient.Client, hash ethcommon.Hash) *ethtypes.Receipt {
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
		panic(err)
	}
	return receipt
}

// getEvmEvent parses the logs in the given receipt and returns the first event that can be parsed
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
