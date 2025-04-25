package main

import (
	"encoding/json"
	"fmt"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcchanneltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
)

// relayFromEvmToSimapp implements the logic of an IBC relayer for a MsgTransfer from EVM roll-up to SimApp.
func relayFromEvmToSimapp(sendPacketEvent *ics26router.ContractSendPacket, proof MptProof, groth16ClientHeight uint64) error {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	msgRecvPacket, err := createMsgRecvPacket(sendPacketEvent, proof, groth16ClientHeight)
	if err != nil {
		return fmt.Errorf("failed to create MsgRecvPacket: %w", err)
	}

	msgRecvPacketResponse, err := utils.BroadcastMessages(clientCtx, sender, 200_000, msgRecvPacket)
	if err != nil {
		return fmt.Errorf("failed to broadcast MsgRecvPacket: %w", err)
	}

	if msgRecvPacketResponse.Code != 0 {
		return fmt.Errorf("failed to execute MsgRecvPacket: %v", msgRecvPacketResponse.RawLog)
	}

	return nil
}

func createMsgRecvPacket(event *ics26router.ContractSendPacket, proof MptProof, groth16ClientHeight uint64) (*ibcchanneltypesv2.MsgRecvPacket, error) {
	transferPayload := event.Packet.Payloads[0]
	ibcPacket := ibcchanneltypesv2.Packet{
		Sequence:          event.Packet.Sequence,
		SourceClient:      event.Packet.SourceClient,
		DestinationClient: event.Packet.DestClient,
		TimeoutTimestamp:  event.Packet.TimeoutTimestamp,
		Payloads: []ibcchanneltypesv2.Payload{
			{
				SourcePort:      transferPayload.SourcePort,
				DestinationPort: transferPayload.DestPort,
				Version:         transferPayload.Version,
				Encoding:        transferPayload.Encoding,
				Value:           transferPayload.Value,
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
		Signer: sender,
	}

	return &msgRecvPacket, nil
}
