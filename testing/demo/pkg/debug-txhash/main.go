package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func getRevertReason(txHash ethcommon.Hash, rpcURL string) []byte {
	client, err := rpc.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}
	var raw json.RawMessage
	err = client.Call(&raw, "debug_traceTransaction", txHash.Hex(), map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to trace transaction: %v", err)
	}
	return raw
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <transaction-hash>", os.Args[0])
	}
	txHash := ethcommon.HexToHash(os.Args[1])
	rpcURL := "http://localhost:8545/"
	result := getRevertReason(txHash, rpcURL)
	fmt.Printf("result %v\n", string(result))
}
