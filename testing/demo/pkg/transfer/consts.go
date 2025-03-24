package main

import "cosmossdk.io/math"

var (
	// transferAmount is the amount of tokens to transfer.
	transferAmount = math.NewInt(100)
)

const (
	// denom is the denomination of the token on SimApp.
	denom = "stake"
	// sender is an address on SimApp that will send funds via the MsgTransfer.
	sender = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"
	// receiver is an address on the EVM chain that will receive funds via the MsgTransfer.
	receiver = "0x7f39c581f595b53c5cb19b5a6e5b8f3a0b1f2f6e"
	// ethPrivateKey is the private key for ethereumAddress.
	ethPrivateKey = "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"

	// tendermintClientID is for the SP1 Tendermint light client on the EVM roll-up.
	tendermintClientID = "07-tendermint-0"
	// groth16ClientID is for the Ethereum light client on the SimApp.
	groth16ClientID = "08-groth16-0"

	// ethereumRPC is the Reth RPC endpoint.
	ethereumRPC = "http://localhost:8545/"
	// celestiaProverRPC is the RPC endpoint for the Celestia prover.
	celestiaProverRPC = "localhost:50051"
)

// ethereumAddress is an address on the EVM chain.
// _ethereumAddress = "0xaF9053bB6c4346381C77C2FeD279B17ABAfCDf4d"
