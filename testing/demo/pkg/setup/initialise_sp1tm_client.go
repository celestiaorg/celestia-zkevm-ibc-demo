package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"math/big"
	"os"
	"os/exec"
	"time"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// groth16ClientID is for the Ethereum light client on the SimApp.
	groth16ClientID = "08-groth16-0"
	// tendermintClientID is for the SP1 Tendermint light client on the EVM roll-up.
	tendermintClientID = "07-tendermint-0"
	// ethPrivateKey is the private key for an account on the EVM roll-up that is funded.
	ethPrivateKey = "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"
	// denom is the denomination of the token on SimApp.
	denom = "stake"
)

var (
	ethChainId   = big.NewInt(80087)
	merklePrefix = [][]byte{[]byte("ibc"), []byte("")}
)

func InitializeTendermintLightClientOnEVMRollup() error {
	// First, check if the Simapp node is healthy before proceeding
	fmt.Println("Checking if simapp node is healthy...")
	if err := utils.CheckNodeHealth("http://localhost:5123", 10); err != nil {
		return fmt.Errorf("simapp node is not healthy, please ensure it is running correctly: %w", err)
	}

	// Check if Ethereum node is healthy
	fmt.Println("Checking if Ethereum node is healthy...")
	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum client: %v", err)
	}

	// Try to get the latest block to verify the node is working
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = ethClient.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("ethereum node is not responding correctly: %v", err)
	}
	fmt.Println("Ethereum node is healthy!")

	// Continue with the existing process
	if err := deployEurekaContracts(); err != nil {
		return err
	}

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Contract Addresses: \n%v\n", addresses)

	if err := addClientOnEVMRollUp(addresses, ethClient); err != nil {
		return err
	}

	if err := registerCounterpartyOnSimapp(); err != nil {
		return err
	}
	return nil
}

// deployEurekaContracts deploys all of the IBC Eureka contracts (including the SP1 ICS07 Tendermint light client contract) on the EVM roll-up.
func deployEurekaContracts() error {
	cmd := exec.Command("forge", "script", "E2ETestDeploy.s.sol:E2ETestDeploy", "--rpc-url", "http://localhost:8545", "--private-key", "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a", "--broadcast")
	cmd.Env = append(cmd.Env, "PRIVATE_KEY=0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a")
	cmd.Dir = "./solidity-ibc-eureka/scripts"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Deploying IBC Eureka smart contracts on the EVM roll-up...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy contracts: %v", err)
	}
	fmt.Println("Deployed IBC Eureka smart contracts on the EVM roll-up.")

	return nil
}

func addClientOnEVMRollUp(addresses utils.ContractAddresses, ethClient *ethclient.Client) error {
	key, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to convert private key: %v", err)
	}

	counterpartyInfo := ics26router.IICS02ClientMsgsCounterpartyInfo{
		ClientId:     groth16ClientID,
		MerklePrefix: merklePrefix,
	}
	tmLightClientAddress := ethcommon.HexToAddress(addresses.ICS07Tendermint)

	router, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return fmt.Errorf("failed to instantiate ICS Core contract: %v", err)
	}

	fmt.Printf("Adding client to the router contract on EVM roll-up...\n")
	tx, err := router.AddClient(GetTransactOpts(key, ethChainId, ethClient), tendermintClientID, counterpartyInfo, tmLightClientAddress)
	if err != nil {
		return fmt.Errorf("failed to add client to router: %v", err)
	}

	receipt := GetTxReceipt(context.Background(), ethClient, tx.Hash())
	event, err := GetEvmEvent(receipt, router.ParseICS02ClientAdded)
	if err != nil {
		return fmt.Errorf("failed to get event: %v", err)
	}
	fmt.Printf("Added client to the router contract on EVM roll-up with clientId %s and counterparty clientId %s\n", event.ClientId, event.CounterpartyInfo.ClientId)

	return nil
}

func registerCounterpartyOnSimapp() error {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	fmt.Printf("Sending bank message...\n")
	bankResp, err := utils.BroadcastMessages(clientCtx, relayer, 500_000, &banktypes.MsgSend{
		FromAddress: relayer,
		ToAddress:   relayer,
		Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(1))),
	})
	if err != nil {
		return fmt.Errorf("failed to send bank message: %v", err)
	}
	fmt.Printf("Sent bank message\n")
	fmt.Printf("Bank response: %v\n", bankResp.Logs)

	fmt.Println("Registering counterparty on simapp...")
	// TODO: this is failing
	resp, err := utils.BroadcastMessages(clientCtx, relayer, 500_000, &clienttypesv2.MsgRegisterCounterparty{
		ClientId:                 groth16ClientID,
		CounterpartyMerklePrefix: merklePrefix,
		CounterpartyClientId:     tendermintClientID,
		Signer:                   relayer,
	})
	if err != nil {
		return fmt.Errorf("failed to register counterparty: %v", err)
	}

	if resp.Code != 0 {
		return fmt.Errorf("failed to register counterparty: %v", resp.RawLog)
	}
	fmt.Println("Registered counterparty on simapp.")
	return nil
}

func GetTransactOpts(key *ecdsa.PrivateKey, chainID *big.Int, ethClient *ethclient.Client) *bind.TransactOpts {
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

func GetTxReceipt(ctx context.Context, ethClient *ethclient.Client, hash ethcommon.Hash) *ethtypes.Receipt {
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
