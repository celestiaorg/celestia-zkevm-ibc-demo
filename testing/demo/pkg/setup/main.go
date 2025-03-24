package main

import (
	"log"
)

func main() {
	err := CreateGroth16LightClient()
	if err != nil {
		log.Fatalf("Failed to create Groth16 light client: %v", err)
	}

	err = CreateTendermintLightClient()
	if err != nil {
		log.Fatalf("Failed to create Tendermint light client: %v", err)
	}

	err = RegisterCounterparty()
	if err != nil {
		log.Fatalf("Failed to register counterparty on simapp: %v", err)
	}
}
