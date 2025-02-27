package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"math/big"
	"os"
	"os/exec"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics02client"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	counterpartyClientId = "channel-0"
	expectedClientId     = "07-tendermint-0"
)

var TendermintLightClientID string

func InitializeSp1TendermintLightClientOnReth() error {
	fmt.Println("Deploying IBC smart contracts on the reth node...")

	if err := runDeploymentCommand(); err != nil {
		return err
	}

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Contract Addresses: \n%v\n", addresses)

	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum client: %v", err)
	}

	if err := createChannelAndCounterpartyOnReth(addresses, ethClient); err != nil {
		return err
	}
	fmt.Println("Created channel and counterparty on reth node.")

	if err := createCounterpartyOnSimapp(); err != nil {
		return err
	}

	fmt.Println("Created counterparty on simapp.")
	return nil

}

// runDeploymentCommand deploys the SP1 ICS07 Tendermint light client contract on the EVM roll-up.
func runDeploymentCommand() error {
	cmd := exec.Command("forge", "script", "E2ETestDeploy.s.sol:E2ETestDeploy", "--rpc-url", "http://localhost:8545", "--private-key", "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a", "--broadcast")
	cmd.Env = append(cmd.Env, "PRIVATE_KEY=0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a")
	cmd.Dir = "./solidity-ibc-eureka/scripts"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy contracts: %v", err)
	}

	return nil
}

func createChannelAndCounterpartyOnReth(addresses utils.ContractAddresses, ethClient *ethclient.Client) error {
	ethChainId := big.NewInt(80087)
	ethPrivateKey := "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"

	icsClientContract, err := ics02client.NewContract(ethcommon.HexToAddress(addresses.ICS02Client), ethClient)
	if err != nil {
		return fmt.Errorf("failed to instantiate ICS Core contract: %v", err)
	}

	counterpartyInfo := ics02client.IICS02ClientMsgsCounterpartyInfo{
		ClientId:     counterpartyClientId,
		MerklePrefix: [][]byte{[]byte("ibc"), []byte("")},
	}

	tmLightClientAddress := ethcommon.HexToAddress(addresses.ICS07Tendermint)

	key, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to convert private key: %v", err)
	}

	fmt.Printf("Adding client to the ICS Client contract on reth node with counterparty clientId %s...\n", counterpartyInfo.ClientId)
	tx, err := icsClientContract.AddClient(GetTransactOpts(key, ethChainId, ethClient), ibcexported.Tendermint, counterpartyInfo, tmLightClientAddress)
	if err != nil {
		return fmt.Errorf("failed to add channel: %v", err)
	}

	receipt := utils.GetTxReceipt(context.Background(), ethClient, tx.Hash())
	event, err := utils.GetEvmEvent(receipt, icsClientContract.ParseICS02ClientAdded)
	if err != nil {
		return fmt.Errorf("failed to get event: %v", err)
	}

	if event.ClientId != expectedClientId {
		return fmt.Errorf("expected clientId %s, got %s", expectedClientId, event.ClientId)
	}

	if event.CounterpartyInfo.ClientId != counterpartyClientId {
		return fmt.Errorf("expected counterparty clientId %s, got %s", counterpartyClientId, event.CounterpartyInfo.ClientId)
	}

	fmt.Printf("Added client to the ICS client contract on reth node with clientId %s and counterparty clientId %s\n", event.ClientId, event.CounterpartyInfo.ClientId)
	TendermintLightClientID = event.CounterpartyInfo.ClientId

	return nil
}

func createCounterpartyOnSimapp() error {
	fmt.Println("Creating counterparty on simapp...")

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	registerCounterPartyResp, err := utils.BroadcastMessages(clientCtx, relayer, 200_000, &channeltypesv2.MsgRegisterCounterparty{
		ChannelId:             counterpartyClientId,
		CounterpartyChannelId: TendermintLightClientID,
		Signer:                relayer,
	})
	if err != nil {
		return fmt.Errorf("failed to register counterparty: %v", err)
	}

	if registerCounterPartyResp.Code != 0 {
		return fmt.Errorf("failed to register counterparty: %v", registerCounterPartyResp.RawLog)
	}

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
