package main

import (
	"log"
)

func main() {
	err := assertVerifierKeys()
	if err != nil {
		log.Fatal("Failed to assert verifier keys: ", err)
	}

	msg, err := createMsgSendPacket()
	if err != nil {
		log.Fatal("Failed to create msg send packet: ", err)
	}

	txHash, err := submitMsgTransfer(msg)
	if err != nil {
		log.Fatal("Failed to submit msg transfer: ", err)
	}

	err = updateTendermintLightClient()
	if err != nil {
		log.Fatalf("Failed to update Tendermint light client: %v\n", err)
	}

	err = relayByTx(txHash, tendermintClientID)
	if err != nil {
		log.Fatalf("Failed to relay IBC transaction: %v", err)
	}

	err = queryBalance()
	if err != nil {
		log.Fatalf("Failed to query balance: %v", err)
	}
}
