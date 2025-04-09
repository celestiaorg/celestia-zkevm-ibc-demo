package main

import (
	"fmt"
	"os"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/sp1ics07tendermint"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

// assertVerifierKeys returns an error if the verifier key on the Tendermint light client does not match the verifier key of the celestia-prover.
func assertVerifierKeys() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("error loading .env file: %v", err)
	}
	if os.Getenv("SP1_PROVER") == "mock" {
		fmt.Printf("Skipping verifier key check because SP1_PROVER=mock\n.")
		return nil
	}

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	lightClient, err := sp1ics07tendermint.NewContract(ethcommon.HexToAddress(addresses.ICS07Tendermint), ethClient)
	if err != nil {
		return err
	}

	// Get the verifier keys from the light client
	clientSTFKey, err := lightClient.UPDATECLIENTPROGRAMVKEY(getCallOpts())
	if err != nil {
		return err
	}
	clientMembershipKey, err := lightClient.MEMBERSHIPPROGRAMVKEY(getCallOpts())
	if err != nil {
		return err
	}

	// Get the verifier keys from the celestia-prover
	proverSTFKey, err := getProverSTFKey()
	if err != nil {
		return err
	}
	proverMembershipKey, err := getProverMembershipKey()
	if err != nil {
		return err
	}

	if clientSTFKey != proverSTFKey {
		return fmt.Errorf("state transition verifier key mismatch. client: %v, prover: %v", clientSTFKey, proverSTFKey)
	}
	if clientMembershipKey != proverMembershipKey {
		return fmt.Errorf("membership verifier key mismatch. client: %v, prover: %v", clientMembershipKey, proverMembershipKey)
	}

	fmt.Printf("The verifier keys on celestia-prover match the verifier keys on the Tendermint light client.\n")
	return nil
}

func getCallOpts() *bind.CallOpts {
	return &bind.CallOpts{
		Pending: false,
		From:    ethcommon.HexToAddress(ethPrivateKey),
	}
}
