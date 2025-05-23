package main

import (
	"context"
	"encoding/binary"
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

// relayByTx implements the logic of an IBC relayer for a MsgTransfer from SimApp to EVM roll-up.
// It processes the sourceTxHash of a MsgTransfer, extracts the IBC events, generates proofs,
// and creates an Ethereum transaction to submit to the ICS26Router contract.
func relayByTx(sourceTxHash string, targetClientID string) error {
	fmt.Printf("Relaying IBC transaction %s to client %s...\n", sourceTxHash, targetClientID)

	event, err := getSendPacketEvent(sourceTxHash)
	if err != nil {
		return fmt.Errorf("failed to get SendPacket event: %w", err)
	}

	resp, err := getCelestiaProverResponse(event)
	if err != nil {
		return err
	}

	privateKey, err := crypto.ToECDSA(ethcommon.FromHex(receiverPrivateKey))
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

	msgRecvPacket, err := getMsgRecvPacket(event, resp)
	if err != nil {
		return fmt.Errorf("failed to get MsgRecvPacket: %w", err)
	}

	fmt.Printf("Submitting RecvPacket transaction...\n")
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
	fmt.Printf("Submitted RecvPacket successfully tx hash %v landed in EVM block %v\n", receipt.TxHash.Hex(), receipt.BlockNumber.Uint64())
	fmt.Printf("Relayed IBC transaction %s to client %s\n", sourceTxHash, targetClientID)
	return nil
}

func getCelestiaProverResponse(event SendPacketEvent) (*proverclient.ProveStateMembershipResponse, error) {
	celestiaProverConn, err := grpc.NewClient(celestiaProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to celestia-prover: %w", err)
	}
	defer celestiaProverConn.Close()
	celestiaProverClient := proverclient.NewProverClient(celestiaProverConn)

	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return nil, err
	}

	path := getPacketCommitmentPath(event)
	fmt.Printf("Packet commitment path: %x\n", path)

	keyPaths := []string{hex.EncodeToString(path)}
	fmt.Printf("Requesting celestia-prover state membership proof key paths %v...\n", keyPaths)
	resp, err := celestiaProverClient.ProveStateMembership(context.Background(), &proverclient.ProveStateMembershipRequest{
		ClientId: addresses.ICS07Tendermint,
		KeyPaths: keyPaths,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get state membership proof: %w", err)
	}
	fmt.Printf("Received celestia-prover state membership proof with height %v.\n", resp.GetHeight())
	return resp, nil
}

func getMsgRecvPacket(event SendPacketEvent, resp *proverclient.ProveStateMembershipResponse) (msgRecvPacket ics26router.IICS26RouterMsgsMsgRecvPacket, err error) {
	// TODO: instead of using getPayloadValue, we should decode the encodedPacketHex and convert the payload into a FungibleTokenPacketData. This isn't trivial to do because there is no utility method to decode the encodedPacketHex and ABI encode it.
	value, err := getPayloadValue()
	if err != nil {
		return ics26router.IICS26RouterMsgsMsgRecvPacket{}, fmt.Errorf("failed to get payload value: %w", err)
	}

	ibcPacket := ics26router.IICS26RouterMsgsPacket{
		Sequence:         event.Sequence,
		SourceClient:     groth16ClientID,
		DestClient:       tendermintClientID,
		TimeoutTimestamp: event.TimeoutTimestamp,
		Payloads: []ics26router.IICS26RouterMsgsPayload{
			{
				SourcePort: transfertypes.PortID,      // transfer
				DestPort:   transfertypes.PortID,      // transfer
				Version:    transfertypes.V1,          // ics20-1
				Encoding:   transfertypes.EncodingABI, // application/x-solidity-abi
				Value:      value,                     // this is ABI encoded fungible token packet data
			},
		},
	}

	return ics26router.IICS26RouterMsgsMsgRecvPacket{
		Packet:          ibcPacket,
		ProofCommitment: resp.Proof,
		ProofHeight: ics26router.IICS02ClientMsgsHeight{
			RevisionNumber: 0,
			RevisionHeight: uint32(resp.Height),
		},
	}, nil
}

type SendPacketEvent struct {
	SourceClient      string
	DestinationClient string
	Sequence          uint64
	TimeoutTimestamp  uint64
	EncodedPacketHex  string
}

func (s SendPacketEvent) String() string {
	return fmt.Sprintf("SourceClient: %s, DestinationClient: %s, Sequence: %d, TimeoutTimestamp: %d, EncodedPacketHex: %s", s.SourceClient, s.DestinationClient, s.Sequence, s.TimeoutTimestamp, s.EncodedPacketHex)
}

func getSendPacketEvent(sourceTxHash string) (SendPacketEvent, error) {
	hash, err := hex.DecodeString(strings.TrimPrefix(sourceTxHash, "0x"))
	if err != nil {
		return SendPacketEvent{}, fmt.Errorf("failed to decode source tx hash: %w", err)
	}

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return SendPacketEvent{}, fmt.Errorf("failed to setup client context: %w", err)
	}

	simAppTx, err := clientCtx.Client.Tx(context.Background(), hash, true)
	if err != nil {
		return SendPacketEvent{}, fmt.Errorf("failed to query transaction: %w", err)
	}

	raw, err := getRawEvent(simAppTx)
	if err != nil {
		return SendPacketEvent{}, err
	}

	sequence, err := strconv.ParseUint(raw["packet_sequence"].(string), 10, 64)
	if err != nil {
		return SendPacketEvent{}, fmt.Errorf("failed to parse packet sequence: %w", err)
	}
	timeoutTimestamp, err := strconv.ParseUint(raw["packet_timeout_timestamp"].(string), 10, 64)
	if err != nil {
		return SendPacketEvent{}, fmt.Errorf("failed to parse timeout timestamp: %w", err)
	}

	return SendPacketEvent{
		SourceClient:      raw["packet_source_client"].(string),
		DestinationClient: raw["packet_dest_client"].(string),
		Sequence:          sequence,
		TimeoutTimestamp:  timeoutTimestamp,
		EncodedPacketHex:  raw["encoded_packet_hex"].(string),
	}, nil
}

// getRawEvent extracts the SendPacket event from the transaction.
func getRawEvent(simAppTx *coretypes.ResultTx) (map[string]interface{}, error) {
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

	if len(sendPacketEvents) == 0 {
		return nil, fmt.Errorf("no SendPacket events found in transaction")
	}
	if len(sendPacketEvents) > 1 {
		return nil, fmt.Errorf("multiple SendPacket events found in transaction")
	}
	sendPacketEvent := sendPacketEvents[0]
	return sendPacketEvent, nil
}

// getPacketCommitmentPath returns the commitment path for the packet.
func getPacketCommitmentPath(event SendPacketEvent) (path []byte) {
	// Convert sequence to big-endian
	sequence := make([]byte, 8)
	binary.BigEndian.PutUint64(sequence, event.Sequence)

	// Create the commitment path according to IBC Eureka specification:
	// 1. Source client ID bytes
	// 2. Marker byte (1 for packet commitment)
	// 3. Sequence number in big-endian
	path = append(path, []byte(event.SourceClient)...)
	path = append(path, byte(1)) // Marker byte for packet commitment
	path = append(path, sequence...)
	return path
}
