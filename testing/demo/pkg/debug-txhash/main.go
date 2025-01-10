package main

import (
	"fmt"
	"log"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func getRevertReason(txHash ethcommon.Hash, rpcURL string) string {
	client, err := rpc.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}
	var result string
	err = client.Call(&result, "debug_traceTransaction", txHash.Hex(), map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to trace transaction: %v", err)
	}
	return result
}

func main() {
	txHash := ethcommon.HexToHash("0x4fa612350e69087839ac4dcb60247d93bc2f739de618bc81829208c21a18a19d")
	rpcURL := "http://localhost:8545/"
	result := getRevertReason(txHash, rpcURL)
	fmt.Printf("result %v\n", result)
}
