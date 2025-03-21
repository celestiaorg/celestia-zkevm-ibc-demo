package main

import (
	"fmt"
	"log"
)

func main() {
	msg, err := createMsgSendPacket()
	if err != nil {
		log.Fatal(err)
	}

	txHash, err := submitMsgTransfer(msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updating Tendermint light client on EVM roll-up...\n")
	err = updateTendermintLightClient()
	if err != nil {
		log.Fatalf("Failed to update Tendermint light client: %v\n", err)
	}
	fmt.Printf("Updated Tendermint light client on EVM roll-up.\n")

	fmt.Printf("Relaying IBC transaction %v...\n", txHash)
	err = relayByTx(txHash, tendermintClientID)
	if err != nil {
		log.Fatalf("Failed to relay IBC transaction: %v", err)
	}
	fmt.Printf("Relayed IBC transaction %v", txHash)
}
