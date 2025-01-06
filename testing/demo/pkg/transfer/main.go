package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cosmossdk.io/math"
	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/client"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// ethereumRPC is the RPC endpoint of the EVM chain.
	ethereumRPC = "http://localhost:8545"

	// celestiaProverEndpoint is the endpoint of the Celestia prover.
	celestiaProverEndpoint = "localhost:50051"

	// channelID is the channel ID on SimApp.
	// TODO: fetch this from the `make setup` command output.
	channelID = "channel-0"

	// clientID is the client ID on SimApp.
	// TODO: fetch this from the `make setup` command output.
	clientID = "08-groth16-0"

	// ics07TMContractAddress is the contract address of the ICS07 light client on the EVM roll-up.
	// TODO: fetch this from the `make setup` command output.
	ics07TMContractAddress = "0x25cdbd2bf399341f8fee22ecdb06682ac81fdc37"
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

	clientHeight, err := QueryLightClientLatestHeight()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	latestHeight, err := QueryLatestHeight()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = GetStateTransitionProof(clientHeight, latestHeight)
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
func QueryLightClientLatestHeight() (latestHeight uint32, err error) {
	fmt.Printf("Querying SP1 ICS07 tendermint light client for the client state's latest height...\n")

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return 0, err
	}

	sp1Ics07Contract, err := sp1ics07tendermint.NewContract(ethcommon.HexToAddress(ics07TMContractAddress), ethClient)
	if err != nil {
		return 0, err
	}
	clientState, err := sp1Ics07Contract.GetClientState(nil)
	if err != nil {
		return 0, err
	}

	fmt.Printf("Client state latest height: %v, revision height %v, revision number %v.\n", clientState.LatestHeight, clientState.LatestHeight.RevisionHeight, clientState.LatestHeight.RevisionNumber)
	return clientState.LatestHeight.RevisionHeight, nil
}

// QueryLatestHeight queries the latest height on SimApp.
func QueryLatestHeight() (height uint32, err error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return 0, err
	}

	node, err := clientCtx.GetNode()
	if err != nil {
		return 0, err
	}

	status, err := node.Status(context.Background())
	if err != nil {
		return 0, err
	}

	return uint32(status.SyncInfo.LatestBlockHeight), nil
}

// GetStateTransitionProof gets the state transition proof from the Celestia prover.
func GetStateTransitionProof(clientHeight uint32, latestHeight uint32) (stateTransitionProof []byte, err error) {
	conn, err := grpc.NewClient(celestiaProverEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to the prover service: %v", err)
	}
	defer conn.Close()

	client := proverclient.NewProverClient(conn)
	request := &proverclient.ProveStateTransitionRequest{
		ClientId: ics07TMContractAddress,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	response, err := client.ProveStateTransition(ctx, request)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to request state transition proof: %w", err)
	}

	fmt.Printf("Proof: %x\n", response.Proof)
	fmt.Printf("Public Values: %x\n", response.PublicValues)
	return response.Proof, nil
}
