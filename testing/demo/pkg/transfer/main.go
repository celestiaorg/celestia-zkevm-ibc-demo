package main

import (
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

	err = updateTendermintLightClient()
	if err != nil {
		log.Fatalf("Failed to update Tendermint light client: %v\n", err)
	}

	err = relayByTx(txHash, tendermintClientID)
	if err != nil {
		log.Fatalf("Failed to relay IBC transaction: %v", err)
	}
}
