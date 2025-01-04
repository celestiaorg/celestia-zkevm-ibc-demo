package main

import (
	"context"
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
	"github.com/cosmos/solidity-ibc-eureka/abigen/sp1ics07tendermint"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// senderAddress is an address on SimApp that will send funds via the MsgTransfer.
	senderAddress = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"

	// receiverAddress is an address on the EVM chain that will receive funds via the MsgTransfer.
	receiverAddress = "0x7f39c581f595b53c5cb19b5a6e5b8f3a0b1f2f6e"

	// denom is the denomination of the token on SimApp.
	denom = "stake"

	// amount is the amount of tokens to transfer.
	amount = 100

	// channelID is the channel ID on SimApp that was created by the `make setup` command.
	channelID = "channel-0"
)

func main() {
	txHash, err := SubmitMsgTransfer()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = QueryPacketCommitments(txHash)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = QueryLightClientLatestHeight()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// TODO
	// Ask the Celestia prover for a state transition proof from the last height (previous step) to the most recent height on SimApp.
	// Ask the Celestia prover for a state membership proof that the receipt is a merkle leaf of the state root.
	// Combine these proofs and packets and submit a MsgUpdateClient and MsgRecvPacket to the EVM rollup.

}

func SubmitMsgTransfer() (txHash string, err error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return "", fmt.Errorf("failed to setup client context: %v", err)
	}

	txHash, err = submitMsgTransfer(clientCtx)
	if err != nil {
		return "", fmt.Errorf("failed to submit MsgTransfer: %v", err)
	}

	return txHash, nil
}

func submitMsgTransfer(clientCtx client.Context) (txHash string, err error) {
	msgTransfer, err := createMsgTransfer()
	if err != nil {
		return "", fmt.Errorf("failed to create MsgTransfer: %w", err)
	}

	fmt.Printf("Broadcasting MsgTransfer...\n")
	response, err := utils.BroadcastMessages(clientCtx, senderAddress, 200_000, &msgTransfer)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast MsgTransfer %w", err)
	}

	if response.Code != 0 {
		return "", fmt.Errorf("failed to execute MsgTransfer %v", response.RawLog)
	}
	fmt.Printf("Broadcasted MsgTransfer. Response code: %v, tx hash: %v\n", response.Code, response.TxHash)
	return response.TxHash, nil
}

func createMsgTransfer() (channeltypesv2.MsgSendPacket, error) {
	coin := sdktypes.NewCoin(denom, math.NewInt(amount))
	transferPayload := ics20lib.ICS20LibFungibleTokenPacketData{
		Denom:    coin.Denom,
		Amount:   coin.Amount.BigInt(),
		Sender:   senderAddress,
		Receiver: receiverAddress,
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
		Signer:           senderAddress,
	}, nil
}

// QueryPacketCommitments queries the packet commitments on the SimApp.
func QueryPacketCommitments(txHash string) error {
	fmt.Printf("Querying packet commitments on SimApp...\n")

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return err
	}

	queryClient := channeltypesv2.NewQueryClient(clientCtx)
	request := channeltypesv2.QueryPacketCommitmentsRequest{
		ChannelId: channelID,
	}
	response, err := queryClient.PacketCommitments(context.Background(), &request)
	if err != nil {
		return fmt.Errorf("failed to query packet commitments: %v", err)
	}

	// TODO what to do with these packets?
	fmt.Printf("Packet commitments: %v\n", response.Commitments)
	return nil
}

// QueryLightClientLatestHeight queries the ICS07 light client on the EVM roll-up for the client state's latest height.
func QueryLightClientLatestHeight() error {
	fmt.Printf("Querying ICS07 light client for the client state's latest height...\n")
	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return err
	}

	// HACKHACK
	// Unfortunately the ICS07 light client on the EVM roll-up doesn't have a fixed contract address. Everytime we deploy it, it appears unique:
	// 0x67cff9B0F9F25c00C71bd8300c3f38553088e234
	lightClient := "0x83b466f5856dc4f531bb5af45045de06889d63cb"
	sp1Ics07Contract, err := sp1ics07tendermint.NewContract(ethcommon.HexToAddress(lightClient), ethClient)
	if err != nil {
		return err
	}
	clientState, err := sp1Ics07Contract.GetClientState(nil)
	if err != nil {
		return err
	}

	fmt.Printf("Client state latest height: %v\n", clientState.LatestHeight)
	return nil
}
