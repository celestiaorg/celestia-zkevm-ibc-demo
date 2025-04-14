package main

import (
	"fmt"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcchanneltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/ethereum/go-ethereum/core/types"
)

// relayByTx implements the logic of an IBC relayer for a MsgTransfer from EVM roll-up to SimApp.
func relayFromEvmToSimapp(sourceTxHash string, targetClientID string) error {
	fmt.Printf("Relaying IBC transaction %s to client %s...\n", sourceTxHash, targetClientID)

	// we already have the mpt proof extracted from the event
	// so we just need to pass it into the function
	// we should probably establish a connection to the simapp chain
	// then we construct the msgRecvPacket and submit it to the simapp chain
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	msgRecvPacket, err := getMsgRecvPacket(event, resp)
	if err != nil {
		return fmt.Errorf("failed to get MsgRecvPacket: %w", err)
	}

	// we can broadcast the msgrecvpacket to the simapp chain
	createClientMsgResponse, err := utils.BroadcastMessages(clientCtx, relayer, 200_000, &clienttypes.MsgCreateClient{
		ClientState:    clientState,
		ConsensusState: consensusState,
		Signer:         relayer,
	})

	cosmosTx, err := getCosmosTx(msgRecvPacket)
	if err != nil {
		return fmt.Errorf("failed to get Cosmos transaction: %w", err)
	}

	fmt.Printf("Cosmos transaction: %v\n", cosmosTx)

	return nil
}

// ethereum event type
func createMsgRecvPacket(event types.Log, commitmentPath []byte) (ibcchanneltypesv2.MsgRecvPacket, error) {
	// we need to construct the msgRecvPacket
	// we need to construct the packet
	ibcPacket := ibcchanneltypesv2.Packet{
		Sequence:          event.Sequence,
		SourceClient:      groth16ClientID,
		DestinationClient: tendermintClientID,
		TimeoutTimestamp:  event.TimeoutTimestamp,
		Payloads: []ibcchanneltypesv2.Payload{
			{
				SourcePort: transfertypes.PortID,
				DestPort:   transfertypes.PortID,
				Version:    transfertypes.V1,
				Encoding:   transfertypes.EncodingABI,
				Value:      commitmentPath,
			},
		},
	}

	msgRecvPacket := ibcchanneltypesv2.MsgRecvPacket{
		Packet: ibcPacket,
	}

	return msgRecvPacket, nil
}
