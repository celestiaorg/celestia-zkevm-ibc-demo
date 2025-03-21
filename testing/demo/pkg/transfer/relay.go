package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// updateTendermintLightClient submits a MsgUpdateClient to the Tendermint light
// client on the EVM roll-up.
func updateTendermintLightClient() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	icsRouter, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
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
	conn, err := grpc.NewClient(celestiaProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Requesting celestia-prover state transition verifier key...\n")
	proverClient := proverclient.NewProverClient(conn)
	info, err := proverClient.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to get celestia-prover info %w", err)
	}
	fmt.Printf("Received celestia-prover state transition verifier key: %v\n", info.StateTransitionVerifierKey)
	verifierKeyDecoded, err := hex.DecodeString(strings.TrimPrefix(info.StateTransitionVerifierKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode verifier key %w", err)
	}
	var verifierKey [32]byte
	copy(verifierKey[:], verifierKeyDecoded)
	fmt.Printf("verifierKey: %x\n", verifierKey)

	fmt.Printf("Requesting celestia-prover state transition proof...\n")
	request := &proverclient.ProveStateTransitionRequest{ClientId: addresses.ICS07Tendermint}
	resp, err := proverClient.ProveStateTransition(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to get state transition proof: %w", err)
	}
	fmt.Printf("Received celestia-prover state transition proof.\n")

	arguments, err := getUpdateClientArguments()
	if err != nil {
		return err
	}

	encoded, err := arguments.Pack(struct {
		Sp1Proof struct {
			VKey         [32]byte
			PublicValues []byte
			Proof        []byte
		}
	}{
		Sp1Proof: struct {
			VKey         [32]byte
			PublicValues []byte
			Proof        []byte
		}{
			VKey:         verifierKey,
			PublicValues: resp.PublicValues,
			Proof:        resp.Proof,
		},
	})
	if err != nil {
		return fmt.Errorf("error packing msg %w", err)
	}

	fmt.Printf("Submitting UpdateClient tx to EVM roll-up...\n")
	fmt.Printf("Client ID: %s\n", tendermintClientID)
	fmt.Printf("Encoded message length: %d bytes\n", len(encoded))
	fmt.Printf("Encoded message: %x\n", encoded)

	tx, err := icsRouter.UpdateClient(getTransactOpts(faucet, eth), tendermintClientID, encoded)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	fmt.Printf("Created transaction with hash: %v and nonce: %v\n", tx.Hash().Hex(), tx.Nonce())

	receipt := getTxReciept(context.Background(), eth, tx.Hash())
	if ethtypes.ReceiptStatusSuccessful != receipt.Status {
		fmt.Printf("Transaction failed with status: %v\n", receipt.Status)
		fmt.Printf("Transaction hash: %s\n", tx.Hash().Hex())
		fmt.Printf("Block number: %d\n", receipt.BlockNumber.Uint64())
		fmt.Printf("Gas used: %d\n", receipt.GasUsed)
		fmt.Printf("Logs: %v\n", receipt.Logs)
	}
	recvBlockNumber := receipt.BlockNumber.Uint64()
	fmt.Printf("Submitted UpdateClient tx in block %v with tx hash %v\n", recvBlockNumber, receipt.TxHash.Hex())
	return nil
}

// relayByTx implements the logic of an IBC relayer.
// It processes source tx, extracts IBC events, generates proofs,
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
	sendPacketEvent, err := getSendPacketEvent(simAppTx)
	if err != nil {
		return fmt.Errorf("failed to get SendPacket event: %w", err)
	}
	packetCommitmentPath, err := getPacketCommitmentPath(sendPacketEvent)
	if err != nil {
		return fmt.Errorf("failed to get packet commitment path: %w", err)
	}
	packetSequenceStr, ok := sendPacketEvent["packet_sequence"].(string)
	if !ok {
		return fmt.Errorf("packet_sequence not found in SendPacket event or not a string")
	}
	packetSequence, err := strconv.ParseUint(packetSequenceStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse packet sequence: %w", err)
	}
	fmt.Printf("Packet sequence: %d\n", packetSequence)

	var resp *proverclient.ProveStateMembershipResponse
	celestiaProverConn, err := grpc.NewClient(celestiaProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to celestia-prover: %w", err)
	}
	defer celestiaProverConn.Close()
	celestiaProverClient := proverclient.NewProverClient(celestiaProverConn)

	fmt.Printf("Requesting celestia-prover state membership proof...\n")
	resp, err = celestiaProverClient.ProveStateMembership(context.Background(), &proverclient.ProveStateMembershipRequest{
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

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}
	ics26Router, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
	if err != nil {
		return err
	}

	timeoutTimestamp, err := strconv.ParseUint(sendPacketEvent["packet_timeout_timestamp"].(string), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse timeout timestamp: %w", err)
	}

	packetHex, ok := sendPacketEvent["encoded_packet_hex"].(string)
	if !ok {
		return fmt.Errorf("encoded_packet_hex not found in SendPacket event or not a string")
	}
	payloadData, err := hex.DecodeString(packetHex)
	if err != nil {
		return fmt.Errorf("failed to decode encoded_packet_hex: %w", err)
	}
	msgRecvPacket := ics26router.IICS26RouterMsgsMsgRecvPacket{
		Packet: ics26router.IICS26RouterMsgsPacket{
			Sequence:         packetSequence,
			SourceClient:     groth16ClientID,
			DestClient:       tendermintClientID,
			TimeoutTimestamp: timeoutTimestamp,
			Payloads: []ics26router.IICS26RouterMsgsPayload{
				{
					SourcePort: transfertypes.PortID,      // transfer
					DestPort:   transfertypes.PortID,      // transfer
					Version:    transfertypes.V1,          // ics20-1
					Encoding:   transfertypes.EncodingABI, // application/x-solidity-abi
					Value:      payloadData,
				},
			},
		},
		ProofCommitment: resp.Proof,
		ProofHeight: ics26router.IICS02ClientMsgsHeight{
			RevisionNumber: 0,
			RevisionHeight: uint32(resp.Height),
		},
	}
	fmt.Printf("msgRecvPacket: %+v\n", msgRecvPacket)

	ethTx, err := ics26Router.RecvPacket(getTransactOpts(privateKey, eth), msgRecvPacket)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	fmt.Printf("Created transaction with hash: %v and nonce: %v\n", ethTx.Hash().Hex(), ethTx.Nonce())
	receipt := getTxReciept(context.Background(), eth, ethTx.Hash())
	fmt.Printf("Transaction sent to Ethereum, hash: %s, status: %d\n", ethTx.Hash().Hex(), receipt.Status)
	return nil
}

func getPacketCommitmentPath(sendPacketEvent map[string]interface{}) ([]byte, error) {
	packetSourceClient, ok := sendPacketEvent["packet_source_client"].(string)
	if !ok {
		return nil, fmt.Errorf("packet_source_client not found in SendPacket event or not a string")
	}
	packetSequenceStr, ok := sendPacketEvent["packet_sequence"].(string)
	if !ok {
		return nil, fmt.Errorf("packet_sequence not found in SendPacket event or not a string")
	}
	packetSequence, err := strconv.ParseUint(packetSequenceStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse packet sequence: %w", err)
	}

	// Convert sequence to big-endian bytes and append
	packetSequenceBigEndian := make([]byte, 8)
	// Store sequence in big-endian format (most significant byte first)
	for i := 7; i >= 0; i-- {
		packetSequenceBigEndian[i] = byte(packetSequence & 0xff)
		packetSequence >>= 8
	}

	// Create the commitment path according to IBC Eureka specification:
	// 1. Source client ID bytes
	// 2. Marker byte (1 for packet commitment)
	// 3. Sequence number in big-endian
	var packetCommitmentPath []byte
	packetCommitmentPath = append(packetCommitmentPath, []byte(packetSourceClient)...)
	packetCommitmentPath = append(packetCommitmentPath, byte(1)) // Marker byte for packet commitment
	packetCommitmentPath = append(packetCommitmentPath, packetSequenceBigEndian...)

	fmt.Printf("packetCommitmentPath: %x\n", packetCommitmentPath)
	return packetCommitmentPath, nil
}

// getSendPacketEvent extracts the SendPacket event from the transaction.
//
// Extracted SendPacket event from transaction: map[
// encoded_packet_hex:0801120c30382d67726f746831362d301a0f30372d74656e6465726d696e742d30208eccebbe062abc040a087472616e7366657212087472616e736665721a0769637332302d31221a6170706c69636174696f6e2f782d736f6c69646974792d6162692a8004000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000000001a000000000000000000000000000000000000000000000000000000000000000057374616b65000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d636f736d6f73316c74767a7077663365673865397337777a6c6571646d7730326c657372646578396a6774307100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a30783766333963353831663539356235336335636231396235613665356238663361306231663266366500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000d74657374207472616e7366657200000000000000000000000000000000000000
// msg_index:0
// packet_dest_client:07-tendermint-0
// packet_sequence:1
// packet_source_client:08-groth16-0
// packet_timeout_timestamp:1742398990]
// sdk.NewEvent(
//
//	types.EventTypeSendPacket,
//	sdk.NewAttribute(types.AttributeKeySrcClient, packet.SourceClient),
//	sdk.NewAttribute(types.AttributeKeyDstClient, packet.DestinationClient),
//	sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
//	sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
//	sdk.NewAttribute(types.AttributeKeyEncodedPacketHex, hex.EncodeToString(encodedPacket)),
//
// ),
func getSendPacketEvent(simAppTx *coretypes.ResultTx) (map[string]interface{}, error) {
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
				packetEvent[key] = value
			}
			sendPacketEvents = append(sendPacketEvents, packetEvent)
		}
	}

	// Check if we found any SendPacket events
	if len(sendPacketEvents) == 0 {
		return nil, fmt.Errorf("no SendPacket events found in transaction")
	}
	if len(sendPacketEvents) > 1 {
		return nil, fmt.Errorf("multiple SendPacket events found in transaction")
	}

	sendPacketEvent := sendPacketEvents[0]
	fmt.Printf("Extracted SendPacket event from transaction: %+v\n", sendPacketEvent)
	return sendPacketEvent, nil
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
	txOpts.GasLimit = 5_000_000

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
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		fmt.Printf("Transaction status: %v\n", receipt.Status)
		fmt.Printf("Transaction hash: %s\n", hash.Hex())
		fmt.Printf("Block number: %d\n", receipt.BlockNumber.Uint64())
		fmt.Printf("Gas used: %d\n", receipt.GasUsed)
		fmt.Printf("Logs: %v\n", receipt.Logs)
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
