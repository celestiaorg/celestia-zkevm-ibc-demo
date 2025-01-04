package main

import (
	"fmt"
	"os"
	"time"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	"github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20lib"
)

const (
	// relayer is an address registered in simapp
	relayer = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"

	// ethereumUserAddress is an address registered in the ethereum chain
	ethereumUserAddress = "0x7f39c581f595b53c5cb19b5a6e5b8f3a0b1f2f6e"

	// denom is the denomination of the token on simapp
	denom = "stake"

	// amount is the amount of tokens to transfer
	amount = 100
)

func main() {
	err := SubmitMsgTransfer()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 2a -> Listen for events that SimApp has emitted that there are pending packets ready to be sent to other chains. It queries the chain for the receipt based on a predetermined location.
	err = QueryPendingPackets()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 2b -> The relayer needs to prove to the EVM rollup that SimApp has actually successfully executed the first part of the transfer: locking up the tokens. Proving this requires two steps: First the relayer queries a state transition proof from the prover process. This will prove the latest state root from the last trusted state root stored in the state of the ICS07 Tendermint smart contract on the EVM. Now the EVM has an up to date record of SimApp's current state (which includes the receipt). Second, the relayer asks the prover for a proof that the receipt is a merkle leaf of the state root i.e. it's part of state
	// 2c -> The prover has a zk circuit for generating both proofs. One takes tendermint headers and uses the SkippingVerification algorithm to assert the latest header. The other takes IAVL merkle proofs and proves some leaf key as part of the root. These are both STARK proofs which can be processed by the smart contracts on the EVM.
	// 2d -> The last step of the relayer is to combine these proofs and packets and submit a MsgUpdateClient and MsgRecvPacket to the EVM rollup.

}

func SubmitMsgTransfer() error {
	fmt.Printf("Setting up client context...\n")
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return fmt.Errorf("failed to setup client context: %v", err)
	}

	err = submitMsgTransfer(clientCtx)
	if err != nil {
		return fmt.Errorf("failed to submit MsgTransfer: %v", err)
	}

	return nil
}

func submitMsgTransfer(clientCtx client.Context) error {
	msgTransfer, err := createMsgTransfer()
	if err != nil {
		return fmt.Errorf("failed to create MsgTransfer: %w", err)
	}

	fmt.Printf("Broadcasting MsgTransfer...\n")
	response, err := utils.BroadcastMessages(clientCtx, relayer, 200_000, &msgTransfer)
	if err != nil {
		return fmt.Errorf("failed to broadcast MsgTransfer %w", err)
	}

	if response.Code != 0 {
		return fmt.Errorf("failed to execute MsgTransfer %v", response.RawLog)
	}
	fmt.Printf("Broadcasted MsgTransfer. Response code: %v, tx hash: %v\n", response.Code, response.TxHash)
	return nil
}

func createMsgTransfer() (channeltypesv2.MsgSendPacket, error) {
	coin := sdktypes.NewCoin(denom, math.NewInt(amount))
	transferPayload := ics20lib.ICS20LibFungibleTokenPacketData{
		Denom:    coin.Denom,
		Amount:   coin.Amount.BigInt(),
		Sender:   relayer,
		Receiver: ethereumUserAddress,
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
		SourceChannel:    ibctesting.FirstChannelID,
		TimeoutTimestamp: uint64(time.Now().Add(30 * time.Minute).Unix()),
		Payloads:         []channeltypesv2.Payload{payload},
		Signer:           relayer,
	}, nil
}

func QueryPendingPackets() error {
	fmt.Println("Querying for pending packets...")
	// TODO: implement
	return nil
}
