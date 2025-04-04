package main

import (
	"fmt"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/ibc/lightclients/groth16"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func updateGroth16LightClient() error {
	fmt.Printf("Updating Groth16 light client on EVM roll-up...\n")

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to get client context: %w", err)
	}

	header := getHeader()
	clientMessage, err := cdctypes.NewAnyWithValue(&header)
	if err != nil {
		return fmt.Errorf("failed to create any value: %w", err)
	}

	resp, err := utils.BroadcastMessages(clientCtx, sender, 200_000, &clienttypes.MsgUpdateClient{
		ClientId:      groth16ClientID,
		ClientMessage: clientMessage,
		Signer:        sender,
	})
	if err != nil {
		return fmt.Errorf("failed to broadcast create client msg: %w", err)
	}
	fmt.Printf("Update client response: %v\n", resp)

	return nil
}

func getHeader() groth16.Header {
	return groth16.Header{
		StateTransitionProof:      []byte{},
		TrustedHeight:             1,
		TrustedCelestiaHeaderHash: []byte{},
		NewStateRoot:              []byte{},
		NewHeight:                 2,
		NewCelestiaHeaderHash:     []byte{},
		DataRoots:                 [][]byte{},
		Timestamp:                 &timestamppb.Timestamp{},
	}
}
