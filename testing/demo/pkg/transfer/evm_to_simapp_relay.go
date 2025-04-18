package main

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcchanneltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
)

// relayByTx implements the logic of an IBC relayer for a MsgTransfer from EVM roll-up to SimApp.
func relayFromEvmToSimapp(sendPacketEvent *ics26router.ContractSendPacket, proof ProofCommitment, groth16ClientHeight uint64) error {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	msgRecvPacket, err := createMsgRecvPacket(sendPacketEvent, proof, groth16ClientHeight)
	if err != nil {
		return fmt.Errorf("failed to create MsgRecvPacket: %w", err)
	}

	// we can broadcast the msgrecvpacket to the simapp chain
	msgRecvPacketResponse, err := utils.BroadcastMessages(clientCtx, cosmosRelayer, 200_000, msgRecvPacket)
	if err != nil {
		return fmt.Errorf("failed to broadcast MsgRecvPacket: %w", err)
	}

	if msgRecvPacketResponse.Code != 0 {
		return fmt.Errorf("failed to execute MsgRecvPacket: %v", msgRecvPacketResponse.RawLog)
	}

	return nil
}

// ethereum event type
func createMsgRecvPacket(event *ics26router.ContractSendPacket, proof ProofCommitment, groth16ClientHeight uint64) (*ibcchanneltypesv2.MsgRecvPacket, error) {
	// TODO: make sure the payload value is correct and compatible with the ibcPacket
	payloadValue, err := getPayloadValueForSimapp(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get payload value: %w", err)
	}
	// event.Packet.Payloads
	ibcPacket := ibcchanneltypesv2.Packet{
		Sequence:          event.Packet.Sequence,
		SourceClient:      event.Packet.SourceClient,
		DestinationClient: event.Packet.DestClient,
		TimeoutTimestamp:  event.Packet.TimeoutTimestamp,
		Payloads: []ibcchanneltypesv2.Payload{
			{
				SourcePort:      transfertypes.PortID,
				DestinationPort: transfertypes.PortID,
				Version:         transfertypes.V1,
				Encoding:        transfertypes.EncodingABI,
				Value:           payloadValue,
			},
		},
	}
	serializedProof, err := json.Marshal(proof)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize proof: %w", err)
	}

	msgRecvPacket := ibcchanneltypesv2.MsgRecvPacket{
		Packet:          ibcPacket,
		ProofCommitment: serializedProof,
		ProofHeight: types.Height{
			RevisionNumber: 0,
			RevisionHeight: groth16ClientHeight,
		},
		Signer: cosmosRelayer,
	}

	return &msgRecvPacket, nil
}

func getPayloadValueForSimapp(event *ics26router.ContractSendPacket) ([]byte, error) {
	// TODO: change to actual transfer amount
	denomNow := "0xCF4fCaC55a3Eb0860Fce5c9328D4F0316F4A6735"
	coin := sdktypes.NewCoin(denomNow, math.NewInt(50))
	transferPayload := transfertypes.FungibleTokenPacketData{
		Denom:    coin.Denom,
		Amount:   coin.Amount.String(),
		Sender:   receiver,
		Receiver: sender,
		Memo:     "transfer back memo",
	}
	payloadValue, err := transfertypes.EncodeABIFungibleTokenPacketData(&transferPayload)
	if err != nil {
		return []byte{}, err
	}
	return payloadValue, nil
}
