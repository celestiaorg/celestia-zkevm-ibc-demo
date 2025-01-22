package main

import (
	"encoding/json"
	"fmt"
	"log"

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
	txHash := ethcommon.HexToHash("0xd69bfa0dc9cd1bf69c2dc38e15a6771354fd387a084e2cb4604f187c55c862a1")
	rpcURL := "http://localhost:8545/"
	result := getRevertReason(txHash, rpcURL)
	fmt.Printf("result %v\n", string(result))
}
