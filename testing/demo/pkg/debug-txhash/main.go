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
	txHash := ethcommon.HexToHash("0xeb18b1724c3a444658ec02efa0a8084425ad8f1b9e891a8e465fec42949bdcf3")
	rpcURL := "http://localhost:8545/"
	result := getRevertReason(txHash, rpcURL)
	fmt.Printf("result %v\n", string(result))
}
