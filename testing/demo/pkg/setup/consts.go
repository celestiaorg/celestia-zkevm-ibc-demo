package main

import "math/big"

const (
	// groth16ClientID is for the Ethereum light client on the SimApp.
	groth16ClientID = "08-groth16-0"
	// tendermintClientID is for the SP1 Tendermint light client on the EVM roll-up.
	tendermintClientID = "07-tendermint-0"
	// ethPrivateKey is the private key for an account on the EVM roll-up that is funded.
	ethPrivateKey = "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"
	// relayer is the address registered on simapp
	relayer = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"
)

var (
	ethChainId   = big.NewInt(80087)
	merklePrefix = [][]byte{[]byte("ibc"), []byte("")}
)
