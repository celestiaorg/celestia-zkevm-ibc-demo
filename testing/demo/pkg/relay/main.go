package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics02client"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/cosmos/solidity-ibc-eureka/abigen/sp1ics07tendermint"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// ethereumRPC is the Reth RPC endpoint.
	ethereumRPC = "http://localhost:8545/"
	// ethPrivateKey is the private key for ethereumAddress.
	ethPrivateKey = "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"
	// rollupClientID is for the SP1 Tendermint light client on the EVM roll-up.
	rollupClientID = "07-tendermint-0"
	// simAppClientID is for the Ethereum light client on the SimApp.
	simAppClientID = "08-groth16-0"

	// ethereumAddress is an address on the EVM chain.
	// ethereumAddress = "0xaF9053bB6c4346381C77C2FeD279B17ABAfCDf4d"
)

func main() {
	fmt.Printf("Updating Tendermint light client on EVM roll-up...\n")
	err := updateTendermintLightClient()
	if err != nil {
		log.Fatalf("Failed to update Tendermint light client: %v\n", err)
	}
	fmt.Printf("Updated Tendermint light client on EVM roll-up.\n")

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <transaction_hash>\n", os.Args[0])
		os.Exit(1)
	}
	txHash := os.Args[1]
	fmt.Printf("Relaying IBC transaction %v...\n", txHash)
	err = relayByTx(txHash, rollupClientID)
	if err != nil {
		log.Fatalf("Failed to relay transaction: %v", err)
	}
	fmt.Printf("Relayed IBC transaction %v", txHash)
}

// updateTendermintLightClient submits a MsgUpdateClient to the Tendermint light client on the EVM roll-up.
func updateTendermintLightClient() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	icsClient, err := ics02client.NewContract(ethcommon.HexToAddress(addresses.ICS02Client), ethClient)
	if err != nil {
		return err
	}
	faucet, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return err
	}
	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, faucet)
	if err != nil {
		return err
	}

	// Connect to the Celestia prover
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Requesting celestia-prover StateTransitionVerifierKey...\n")
	proverClient := proverclient.NewProverClient(conn)
	info, err := proverClient.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to get celestia-prover info %w", err)
	}
	fmt.Printf("Received celestia-prover StateTransitionVerifierKey: %v\n", info.StateTransitionVerifierKey)
	verifierKeyDecoded, err := hex.DecodeString(strings.TrimPrefix(info.StateTransitionVerifierKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode verifier key %w", err)
	}
	var verifierKey [32]byte
	copy(verifierKey[:], verifierKeyDecoded)

	fmt.Printf("Requesting celestia-prover state transition proof...\n")
	request := &proverclient.ProveStateTransitionRequest{ClientId: addresses.ICS07Tendermint}
	// Get state transition proof from Celestia prover with retry logic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var resp *proverclient.ProveStateTransitionResponse
	for retries := 0; retries < 3; retries++ {
		resp, err = proverClient.ProveStateTransition(ctx, request)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled while getting state transition proof: %w", ctx.Err())
		}
		time.Sleep(time.Second * time.Duration(retries+1))
	}
	if err != nil {
		return fmt.Errorf("failed to get state transition proof after retries: %w", err)
	}
	fmt.Printf("Received celestia-prover state transition proof.\n")
	arguments, err := getUpdateClientArguments()
	if err != nil {
		return err
	}
	encoded, err := arguments.Pack(sp1ics07tendermint.IUpdateClientMsgsMsgUpdateClient{
		Sp1Proof: sp1ics07tendermint.ISP1MsgsSP1Proof{
			VKey:         verifierKey,
			PublicValues: resp.PublicValues,
			Proof:        resp.Proof,
		},
	})
	if err != nil {
		return fmt.Errorf("error packing msg %w", err)
	}

	fmt.Printf("Submitting UpdateClient tx to EVM roll-up...\n")
	tx, err := icsClient.UpdateClient(getTransactOpts(faucet, eth), rollupClientID, encoded)
	if err != nil {
		return err
	}
	receipt := getTxReciept(context.Background(), eth, tx.Hash())
	if ethtypes.ReceiptStatusSuccessful != receipt.Status {
		return fmt.Errorf("receipt status want %v, got %v. logs: %v", ethtypes.ReceiptStatusSuccessful, receipt.Status, receipt.Logs)
	}
	recvBlockNumber := receipt.BlockNumber.Uint64()
	fmt.Printf("Submitted UpdateClient tx in block %v with tx hash %v\n", recvBlockNumber, receipt.TxHash.Hex())
	return nil
}

func getTransactOpts(key *ecdsa.PrivateKey, chain ethereum.Ethereum) *bind.TransactOpts {
	ethClient, err := ethclient.Dial(chain.RPC)
	if err != nil {
		log.Fatal(err)
	}

	fromAddress := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		nonce = 0
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(key, chain.ChainID)
	if err != nil {
		log.Fatal(err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.GasPrice = gasPrice

	// Set a specific gas limit
	txOpts.GasLimit = 3000000 // Example gas limit; adjust as needed

	return txOpts
}

func getTxReciept(ctx context.Context, chain ethereum.Ethereum, hash ethcommon.Hash) *ethtypes.Receipt {
	ethClient, err := ethclient.Dial(chain.RPC)
	if err != nil {
		log.Fatal(err)
	}

	var receipt *ethtypes.Receipt
	err = utils.WaitForCondition(time.Second*30, time.Second, func() (bool, error) {
		receipt, err = ethClient.TransactionReceipt(ctx, hash)
		if err != nil {
			return false, nil
		}

		return receipt != nil, nil
	})
	if err != nil {
		log.Fatalf("Failed to fetch receipt: %v", err)
	}

	// Log more details about the receipt
	// fmt.Printf("Transaction hash: %s\n", hash.Hex())
	// fmt.Printf("Block number: %d\n", receipt.BlockNumber.Uint64())
	// fmt.Printf("Gas used: %d\n", receipt.GasUsed)
	// fmt.Printf("Logs: %v\n", receipt.Logs)
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		fmt.Println("Transaction failed. Inspect logs or contract.")
	}

	return receipt
}

func getUpdateClientArguments() (abi.Arguments, error) {
	var updateClientABI = "[{\"type\":\"function\",\"name\":\"updateClient\",\"stateMutability\":\"pure\",\"inputs\":[{\"name\":\"o3\",\"type\":\"tuple\",\"internalType\":\"struct IUpdateClientMsgs.MsgUpdateClient\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"struct ISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]}],\"outputs\":[]}]"

	parsed, err := abi.JSON(strings.NewReader(updateClientABI))
	if err != nil {
		return nil, err
	}

	return parsed.Methods["updateClient"].Inputs, nil
}

// relayByTx implements the logic that the relayer would perform directly
// It processes source transactions, extracts IBC events, generates proofs,
// and creates an Ethereum transaction to submit to the ICS26Router contract.
func relayByTx(sourceTxHash string, targetClientID string) error {
	fmt.Printf("Relaying transaction %s to client %s...\n", sourceTxHash, targetClientID)

	txID, err := hex.DecodeString(strings.TrimPrefix(sourceTxHash, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode source tx hash: %w", err)
	}

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %w", err)
	}
	fmt.Printf("Querying transaction and extracting IBC events...\n")
	simAppTx, err := clientCtx.Client.Tx(context.Background(), txID, true)
	if err != nil {
		return fmt.Errorf("failed to query transaction: %w", err)
	}
	fmt.Printf("Queried transaction and extracted %v events.\n", len(simAppTx.TxResult.Events))

	// Extract the SendPacket events from the transaction
	var sendPacketEvents []map[string]interface{}
	for _, event := range simAppTx.TxResult.Events {
		// Check if this is a SendPacket event
		if event.Type == "send_packet" {
			// Extract the event attributes
			packetEvent := make(map[string]interface{})
			for _, attr := range event.Attributes {
				key := string(attr.Key)
				value := string(attr.Value)

				switch key {
				case "packet_src_port", "packet_src_channel", "packet_dst_port", "packet_dst_channel", "packet_data", "packet_sequence", "packet_timeout_timestamp":
					// Store string values as is
					packetEvent[key] = value
				default:
					// Store any other attributes
					packetEvent[key] = value
				}
			}
			sendPacketEvents = append(sendPacketEvents, packetEvent)
		}
	}

	// Check if we found any SendPacket events
	if len(sendPacketEvents) == 0 {
		return fmt.Errorf("no SendPacket events found in transaction")
	}
	if len(sendPacketEvents) > 1 {
		return fmt.Errorf("multiple SendPacket events found in transaction")
	}

	sendPacketEvent := sendPacketEvents[0]
	fmt.Printf("Extracted SendPacket event from transaction: %+v\n", sendPacketEvent)

	// Generate a proof for the packet commitment
	// This would involve:
	// 1. Determining the key path in the SimApp state tree where the packet commitment is stored
	// 2. Getting a proof for that key from the prover

	// Parse the packet sequence as uint64
	packetSequenceStr, ok := sendPacketEvent["packet_sequence"].(string)
	if !ok {
		return fmt.Errorf("packet_sequence not found in SendPacket event or not a string")
	}
	packetSequence, err := strconv.ParseUint(packetSequenceStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse packet sequence: %w", err)
	}
	fmt.Printf("Packet sequence: %d\n", packetSequence)

	// Create the commitment path according to IBC Eureka specification:
	// - Source client ID bytes
	// - Marker byte (1 for packet commitment)
	// - Sequence number in big-endian
	var packetCommitmentPath []byte

	// TODO: the version of ibc-go that SimApp uses doesn't emit the source client ID in the SendPacket event.
	// After we upgrade ibc-go, stop hard-coding the simAppClientID and fetch the event from the packet.
	packetCommitmentPath = append(packetCommitmentPath, []byte(simAppClientID)...)
	packetCommitmentPath = append(packetCommitmentPath, byte(1)) // Marker byte for packet commitment

	// Convert sequence to big-endian bytes and append
	sequenceBytes := make([]byte, 8)
	// Store sequence in big-endian format (most significant byte first)
	for i := 7; i >= 0; i-- {
		sequenceBytes[i] = byte(packetSequence & 0xff)
		packetSequence >>= 8
	}
	packetCommitmentPath = append(packetCommitmentPath, sequenceBytes...)
	fmt.Printf("Generating proof for packet commitment with path: %x\n", packetCommitmentPath)

	celestiaProverConn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to celestia-prover: %w", err)
	}
	defer celestiaProverConn.Close()
	celestiaProverClient := proverclient.NewProverClient(celestiaProverConn)

	fmt.Printf("Requesting celestia-prover state membership proof...\n")
	resp, err := celestiaProverClient.ProveStateMembership(context.Background(), &proverclient.ProveStateMembershipRequest{
		Height:   simAppTx.Height,
		KeyPaths: []string{hex.EncodeToString(packetCommitmentPath)},
	})
	if err != nil {
		return fmt.Errorf("failed to get state membership proof: %w", err)
	}
	fmt.Printf("Received celestia-prover state membership proof with height %v.\n", resp.GetHeight())

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return fmt.Errorf("failed to get contract addresses: %w", err)
	}

	ics26RouterAddr := ethcommon.HexToAddress(addresses.ICS26Router)
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ics26Router, err := ics26router.NewContract(ics26RouterAddr, ethClient)
	if err != nil {
		return err
	}
	timeoutTimestamp, err := strconv.ParseUint(sendPacketEvent["packet_timeout_timestamp"].(string), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse timeout timestamp: %w", err)
	}

	payloadData, err := hex.DecodeString(sendPacketEvent["packet_data"].(string))
	if err != nil {
		return fmt.Errorf("failed to decode payload data: %w", err)
	}

	ethTx, err := ics26Router.RecvPacket(getTransactOpts(privateKey, eth), ics26router.IICS26RouterMsgsMsgRecvPacket{
		Packet: ics26router.IICS26RouterMsgsPacket{
			Sequence:         uint32(packetSequence),
			SourceClient:     simAppClientID,
			DestClient:       targetClientID,
			TimeoutTimestamp: timeoutTimestamp,
			Payloads: []ics26router.IICS26RouterMsgsPayload{
				{
					SourcePort: "",                                           // There are no ports in IBC Eureka
					DestPort:   "",                                           // There are no ports in IBC Eureka
					Version:    sendPacketEvent["payload_version"].(string),  // ics20-1
					Encoding:   sendPacketEvent["payload_encoding"].(string), // application/x-solidity-abi
					Value:      payloadData,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	fmt.Printf("Created transaction: %v\n", ethTx.Hash().Hex())

	err = ethClient.SendTransaction(context.Background(), ethTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}
	receipt := getTxReciept(context.Background(), eth, ethTx.Hash())
	fmt.Printf("Transaction sent to Ethereum, hash: %s, status: %d\n", ethTx.Hash().Hex(), receipt.Status)
	return nil
}
