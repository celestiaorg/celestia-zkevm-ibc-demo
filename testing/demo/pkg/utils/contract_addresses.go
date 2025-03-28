package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

type ContractAddresses struct {
	ERC20           string `json:"erc20"`
	ICS07Tendermint string `json:"ics07Tendermint"`
	ICS20Transfer   string `json:"ics20Transfer"`
	ICS26Router     string `json:"ics26Router"`
	IBCERC20        string `json:"ibcERC20"`
}

func (c ContractAddresses) String() string {
	return fmt.Sprintf("ERC20: %s\nICS07Tendermint: %s\nICS20Transfer: %s\nICS26Router: %s\nIBCERC20: %s\n", c.ERC20, c.ICS07Tendermint, c.ICS20Transfer, c.ICS26Router, c.IBCERC20)
}

type Transaction struct {
	ContractName    string `json:"contractName"`
	ContractAddress string `json:"contractAddress"`
}

type RunLatest struct {
	Transactions []Transaction `json:"transactions"`
}

func ExtractDeployedContractAddresses() (ContractAddresses, error) {
	filePath := "./solidity-ibc-eureka/broadcast/E2ETestDeploy.s.sol/80087/run-latest.json"
	file, err := os.ReadFile(filePath)
	if err != nil {
		return ContractAddresses{}, fmt.Errorf("error reading file: %v", err)
	}

	var runLatest RunLatest
	if err := json.Unmarshal(file, &runLatest); err != nil {
		return ContractAddresses{}, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	addresses := ContractAddresses{}
	for _, tx := range runLatest.Transactions {
		switch tx.ContractName {
		case "IBCERC20":
			addresses.IBCERC20 = tx.ContractAddress
		case "SP1ICS07Tendermint":
			addresses.ICS07Tendermint = tx.ContractAddress
		case "ICS20Transfer":
			addresses.ICS20Transfer = tx.ContractAddress
		case "ICS26Router":
			addresses.ICS26Router = tx.ContractAddress
		}
	}

	if addresses.IBCERC20 == "" {
		return ContractAddresses{}, fmt.Errorf("IBCERC20 contract address not found in deployment file")
	}

	return addresses, nil
}
