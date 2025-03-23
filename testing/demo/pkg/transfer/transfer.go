package main

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// createMsgSendPacket returns a msg that sends 100stake over IBC.
func createMsgSendPacket() (channeltypesv2.MsgSendPacket, error) {
	coin := sdktypes.NewCoin(denom, math.NewInt(100))
	transferPayload := transfertypes.FungibleTokenPacketData{
		Denom:    coin.Denom,
		Amount:   coin.Amount.String(),
		Sender:   sender,
		Receiver: receiver,
		Memo:     "test transfer",
	}
	transferBz, err := transfertypes.EncodeABIFungibleTokenPacketData(&transferPayload)
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
		SourceClient:     groth16ClientID,
		TimeoutTimestamp: uint64(time.Now().Add(30 * time.Minute).Unix()),
		Payloads:         []channeltypesv2.Payload{payload},
		Signer:           sender,
	}, nil
}

func submitMsgTransfer(msg channeltypesv2.MsgSendPacket) (txHash string, err error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return "", fmt.Errorf("failed to setup client context: %v", err)
	}

	fmt.Printf("Broadcasting MsgTransfer...\n")
	response, err := utils.BroadcastMessages(clientCtx, sender, 200_000, &msg)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast MsgTransfer %w", err)
	}

	if response.Code != 0 {
		return "", fmt.Errorf("failed to execute MsgTransfer %v with status code %v", response.RawLog, response.Code)
	}
	fmt.Printf("Broadcast MsgTransfer success. TxHash: %v landed in height %v\n", response.TxHash, response.Height)
	return response.TxHash, nil
}
