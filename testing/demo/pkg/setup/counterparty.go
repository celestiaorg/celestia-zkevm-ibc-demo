package main

import (
	"fmt"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
)

// RegisterCounterparty registers the counterparty on simapp. This connects the
// Groth16 light client on simapp with the Tendermint light client on the EVM
// roll-up.
func RegisterCounterparty() error {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	fmt.Println("Registering counterparty on simapp...")
	resp, err := utils.BroadcastMessages(clientCtx, relayer, 500_000, &clienttypesv2.MsgRegisterCounterparty{
		ClientId:                 groth16ClientID,
		CounterpartyMerklePrefix: merklePrefix,
		CounterpartyClientId:     tendermintClientID,
		Signer:                   relayer,
	})
	if err != nil {
		return fmt.Errorf("failed to register counterparty on simapp: %v", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("failed to register counterparty on simapp: %v", resp.RawLog)
	}
	fmt.Println("Registered counterparty on simapp.")
	return nil
}
