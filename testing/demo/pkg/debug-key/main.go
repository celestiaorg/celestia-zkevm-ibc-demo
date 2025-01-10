package main

import (
	"crypto/ecdsa"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// Replace with your private key
	privateKeyHex := "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"

	// Parse the private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Derive the public key
	publicKey := privateKey.Public().(*ecdsa.PublicKey)

	// Compute the Ethereum address
	address := crypto.PubkeyToAddress(*publicKey)
	fmt.Printf("Derived Ethereum address: %s\n", address.Hex())
}
