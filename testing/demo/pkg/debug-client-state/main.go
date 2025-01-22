package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func main() {
	// clientState is copied from scripts/genesis.json
	const clientStateHex = "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005000000000000000000000000000000000000000000000000000000000012750000000000000000000000000000000000000000000000000000000000001baf8000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000a7a6b6962632d64656d6f00000000000000000000000000000000000000000000"

	clientStateBytes, err := hex.DecodeString(clientStateHex)
	if err != nil {
		log.Fatalf("Failed to decode clientState hex: %v", err)
	}

	// ABI JSON string
	abiJSON := `[{
		"type": "function",
		"name": "getClientState",
		"inputs": [],
		"outputs": [{
			"name": "",
			"type": "tuple",
			"components": [
				{ "name": "chainId", "type": "string" },
				{ "name": "trustLevel", "type": "tuple", "components": [
					{ "name": "numerator", "type": "uint8" },
					{ "name": "denominator", "type": "uint8" }
				]},
				{ "name": "latestHeight", "type": "tuple", "components": [
					{ "name": "revisionNumber", "type": "uint32" },
					{ "name": "revisionHeight", "type": "uint32" }
				]},
				{ "name": "trustingPeriod", "type": "uint32" },
				{ "name": "unbondingPeriod", "type": "uint32" },
				{ "name": "isFrozen", "type": "bool" },
				{ "name": "zkAlgorithm", "type": "uint8" }
			]
		}]
	}]`

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v", err)
	}

	fmt.Println("ABI successfully parsed")

	// var clientState sp1ics07tendermint.IICS07TendermintMsgsClientState
	clientState, err := parsedABI.Unpack("getClientState", clientStateBytes)
	if err != nil {
		log.Fatalf("Failed to unpack clientState: %v", err)
	}

	fmt.Printf("Decoded clientState: %#v\n", clientState)

	parsedABI.Pack("getClientState")
	parsedABI.Pack()
	// clientState
}

// Decoded clientState: []interface {}{struct { ChainId string "json:\"chainId\""; TrustLevel struct { Numerator uint8 "json:\"numerator\""; Denominator uint8 "json:\"denominator\"" } "json:\"trustLevel\""; LatestHeight struct { RevisionNumber uint32 "json:\"revisionNumber\""; RevisionHeight uint32 "json:\"revisionHeight\"" } "json:\"latestHeight\""; TrustingPeriod uint32 "json:\"trustingPeriod\""; UnbondingPeriod uint32 "json:\"unbondingPeriod\""; IsFrozen bool "json:\"isFrozen\""; ZkAlgorithm uint8 "json:\"zkAlgorithm\"" }
// {ChainId:"zkibc-demo", TrustLevel:struct { Numerator uint8 "json:\"numerator\""; Denominator uint8 "json:\"denominator\"" }{Numerator:0x1, Denominator:0x3}, LatestHeight:struct { RevisionNumber uint32 "json:\"revisionNumber\""; RevisionHeight uint32 "json:\"revisionHeight\"" }{RevisionNumber:0x0, RevisionHeight:0x5}, TrustingPeriod:0x127500, UnbondingPeriod:0x1baf80, IsFrozen:false, ZkAlgorithm:0x1}}

// sp1ics07tendermint.IICS07TendermintMsgsClientStateABI
// {
//     "type": "function",
//     "name": "getClientState",
//     "inputs": [],
//     "outputs": [
//       {
//         "name": "",
//         "type": "tuple",
//         "internalType": "struct IICS07TendermintMsgs.ClientState",
//         "components": [
//           {
//             "name": "chainId",
//             "type": "string",
//             "internalType": "string"
//           },
//           {
//             "name": "trustLevel",
//             "type": "tuple",
//             "internalType": "struct IICS07TendermintMsgs.TrustThreshold",
//             "components": [
//               {
//                 "name": "numerator",
//                 "type": "uint8",
//                 "internalType": "uint8"
//               },
//               {
//                 "name": "denominator",
//                 "type": "uint8",
//                 "internalType": "uint8"
//               }
//             ]
//           },
//           {
//             "name": "latestHeight",
//             "type": "tuple",
//             "internalType": "struct IICS02ClientMsgs.Height",
//             "components": [
//               {
//                 "name": "revisionNumber",
//                 "type": "uint32",
//                 "internalType": "uint32"
//               },
//               {
//                 "name": "revisionHeight",
//                 "type": "uint32",
//                 "internalType": "uint32"
//               }
//             ]
//           },
//           {
//             "name": "trustingPeriod",
//             "type": "uint32",
//             "internalType": "uint32"
//           },
//           {
//             "name": "unbondingPeriod",
//             "type": "uint32",
//             "internalType": "uint32"
//           },
//           {
//             "name": "isFrozen",
//             "type": "bool",
//             "internalType": "bool"
//           },
//           {
//             "name": "zkAlgorithm",
//             "type": "uint8",
//             "internalType": "enum ISP1Msgs.SupportedZkAlgorithm"
//           }
//         ]
//       }
//     ],
//     "stateMutability": "view"
//   },
