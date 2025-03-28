package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ContractAddresses struct {
	ERC20           string `json:"erc20"`
	ICS07Tendermint string `json:"ics07Tendermint"`
	ICS20Transfer   string `json:"ics20Transfer"`
	ICS26Router     string `json:"ics26Router"`
	IBCERC20Logic   string `json:"ibcERC20Logic"`
}

func (c ContractAddresses) String() string {
	return fmt.Sprintf("ERC20: %s\nICS07Tendermint: %s\nICS20Transfer: %s\nICS26Router: %s\nIBCERC20Logic: %s\n", c.ERC20, c.ICS07Tendermint, c.ICS20Transfer, c.ICS26Router, c.IBCERC20Logic)
}

func ExtractDeployedContractAddresses() (ContractAddresses, error) {
	filePath := "./solidity-ibc-eureka/broadcast/E2ETestDeploy.s.sol/80087/run-latest.json"
	file, err := os.ReadFile(filePath)
	if err != nil {
		return ContractAddresses{}, fmt.Errorf("error reading file: %v", err)
	}

	var runLatest map[string]interface{}
	if err := json.Unmarshal(file, &runLatest); err != nil {
		return ContractAddresses{}, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	returns, ok := runLatest["returns"].(map[string]interface{})
	if !ok {
		return ContractAddresses{}, fmt.Errorf("no valid returns found")
	}

	returnValue, ok := returns["0"].(map[string]interface{})
	if !ok {
		return ContractAddresses{}, fmt.Errorf("no valid return value found")
	}

	value, ok := returnValue["value"].(string)
	if !ok {
		return ContractAddresses{}, fmt.Errorf("no valid value found")
	}

	unescapedValue := strings.ReplaceAll(value, "\\\"", "\"")

	var addresses ContractAddresses
	if err := json.Unmarshal([]byte(unescapedValue), &addresses); err != nil {
		return ContractAddresses{}, fmt.Errorf("error unmarshalling contract addresses: %v", err)
	}

	return addresses, nil
}
