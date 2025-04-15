package main

import (
	"encoding/json"
	"fmt"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	// transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcchanneltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	// "github.com/ethereum/go-ethereum/core/types"
)

type Proof struct {
	Proof  []byte
	MptKey []byte
	Height uint64
}

// relayByTx implements the logic of an IBC relayer for a MsgTransfer from EVM roll-up to SimApp.
func relayFromEvmToSimapp(sendPacketEvent *ics26router.ContractSendPacket, proof Proof) error {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	msgRecvPacket, err := createMsgRecvPacket(sendPacketEvent, proof)
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

	// if err != nil {
	// 	return fmt.Errorf("failed to get Cosmos transaction: %w", err)
	// }

	return nil
}

// ethereum event type
func createMsgRecvPacket(event *ics26router.ContractSendPacket, proof Proof) (*ibcchanneltypesv2.MsgRecvPacket, error) {
	fmt.Printf("event payload value: %v\n", event.Packet.Payloads[0].Value)
	// TODO: make sure the payload value is correct and compatible with the ibcPacket
	fmt.Println("source client: ", event.Packet.SourceClient)
	fmt.Println("destination client: ", event.Packet.DestClient)
	payloadValue := event.Packet.Payloads[0].Value
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
	serializedProof, err := serializeProof(proof.Proof, proof.MptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize proof: %w", err)
	}

	msgRecvPacket := ibcchanneltypesv2.MsgRecvPacket{
		Packet:          ibcPacket,
		ProofCommitment: serializedProof,
		ProofHeight: types.Height{
			RevisionNumber: 0,
			RevisionHeight: proof.Height,
		},
		Signer: cosmosRelayer,
	}

	return &msgRecvPacket, nil
}

func serializeProof(mptProof []byte, mptKey []byte) ([]byte, error) {
	proof := Proof{
		Proof:  mptProof,
		MptKey: mptKey,
	}

	return json.Marshal(proof)
}

func deserializeProof(serializedProof []byte) (Proof, error) {
	var proof Proof
	err := json.Unmarshal(serializedProof, &proof)
	if err != nil {
		return Proof{}, fmt.Errorf("failed to deserialize proof: %w", err)
	}
	return proof, nil
}