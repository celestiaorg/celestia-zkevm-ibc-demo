package main

import (
	"fmt"
	"log"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
)

func main() {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		log.Fatalf("failed to get contract addresses: %v", err)
	}
	fmt.Printf("Contract addresses: \n%v\n", addresses)
}
