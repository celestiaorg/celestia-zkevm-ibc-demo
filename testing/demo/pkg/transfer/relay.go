package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// relayByTx implements the logic of an IBC relayer.
// It processes source tx, extracts IBC events, generates proofs,
// and creates an Ethereum transaction to submit to the ICS26Router contract.
func relayByTx(sourceTxHash string, targetClientID string) error {
	fmt.Printf("Relaying IBC transaction %s to client %s...\n", sourceTxHash, targetClientID)

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
	receipt, err := getTxReciept(context.Background(), eth, ethTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("RecvPacket failed with status: %v tx hash: %s block number: %d gas used: %d logs: %v", receipt.Status, receipt.TxHash.Hex(), receipt.BlockNumber.Uint64(), receipt.GasUsed, receipt.Logs)
	}
	fmt.Printf("RecvPacket success in block %v\n", receipt.BlockNumber.Uint64())
	fmt.Printf("Relayed IBC transaction %s to client %s...\n", sourceTxHash, targetClientID)
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
