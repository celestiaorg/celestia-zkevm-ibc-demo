package main

import (
	"fmt"
	"log"
	"time"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20lib"
)

const (
	// sender is an address on SimApp that will send funds via the MsgTransfer.
	Sender = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"
	// receiver is an address on the EVM chain that will receive funds via the MsgTransfer.
	Receiver = "0x7f39c581f595b53c5cb19b5a6e5b8f3a0b1f2f6e"
	// denom is the denomination of the token on SimApp.
	Denom = "stake"
	Amount = 100
	// sourceChannel is hard-coded to the name used by the first channel.
	SourceChannel = ibctesting.FirstChannelID
	// SenderInitialBalance is the initial balance of the sender from genesis.
	SenderInitialBalance = 274883996352
	// ReceiverInitialBalance is the initial balance of the receiver.
	ReceiverInitialBalance = 0
)

func main() {
	msg, err := createMsgSendPacket()
	if err != nil {
		log.Fatal(err)
	}

	_, err = submitMsgTransfer(msg)
	if err != nil {
		log.Fatal(err)
	}

}

// createMsgSendPacket returns a msg that sends 100stake over IBC.
func createMsgSendPacket() (channeltypesv2.MsgSendPacket, error) {
	coin := sdktypes.NewCoin(Denom, math.NewInt(Amount))
	transferPayload := ics20lib.ICS20LibFungibleTokenPacketData{
		Denom:    coin.Denom,
		Amount:   coin.Amount.BigInt(),
		Sender:   Sender,
		Receiver: Receiver,
		Memo:     "test transfer",
	}
	transferBz, err := ics20lib.EncodeFungibleTokenPacketData(transferPayload)
	if err != nil {
		return channeltypesv2.MsgSendPacket{}, err
	}
	payload := channeltypesv2.Payload{
		SourcePort:      transfertypes.PortID,
		DestinationPort: transfertypes.PortID,
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingABI,
		Value:           transferBz,
	}

	return channeltypesv2.MsgSendPacket{
		SourceChannel:    SourceChannel,
		TimeoutTimestamp: uint64(time.Now().Add(30 * time.Minute).Unix()),
		Payloads:         []channeltypesv2.Payload{payload},
		Signer:           Sender,
	}, nil
}

func submitMsgTransfer(msg channeltypesv2.MsgSendPacket) (txHash string, err error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return "", fmt.Errorf("failed to setup client context: %v", err)
	}

	fmt.Printf("Broadcasting MsgTransfer...\n")
	response, err := utils.BroadcastMessages(clientCtx, Sender, 200_000, &msg)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast MsgTransfer %w", err)
	}

	if response.Code != 0 {
		return "", fmt.Errorf("failed to execute MsgTransfer %v", response.RawLog)
	}
	fmt.Printf("Broadcasted MsgTransfer. Response code: %v, tx hash: %v\n", response.Code, response.TxHash)
	return response.TxHash, nil
}
