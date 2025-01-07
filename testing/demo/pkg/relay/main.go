package main

import (
	"context"
	"fmt"
	"log"
	"time"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/sp1ics07tendermint"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// ethereumRPC is the RPC endpoint of the EVM chain.
	ethereumRPC = "http://localhost:8545"
	// celestiaProverEndpoint is the endpoint of the Celestia prover.
	celestiaProverEndpoint = "localhost:50051"
	// channelID is the channel ID on SimApp.
	// TODO: fetch this from the `make setup` command output.
	channelID = "channel-0"
	// ics07TMContractAddress is the contract address of the ICS07 light client on the EVM roll-up.
	// TODO: fetch this from the `make setup` command output.
	ics07TMContractAddress = "0x25cdbd2bf399341f8fee22ecdb06682ac81fdc37"
)

func main() {
	// Ask the Celestia prover for a state transition proof.
	_, err := GetStateTransitionProof()
	if err != nil {
		log.Fatal(err)
	}

	packetResp, err := QueryPacketCommitments()
	if err != nil {
		log.Fatal(err)
	}

	// Ask the Celestia prover for a state membership proof that the packet
	// commitments are part of the state root at a particular block height.
	_, err = GetMembershipProof(packetResp)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: combine these proofs and packets and submit a MsgUpdateClient and
	// MsgRecvPacket to the EVM rollup.
}

// QueryPacketCommitments queries the packet commitments on the SimApp.
func QueryPacketCommitments() (*channeltypesv2.QueryPacketCommitmentsResponse, error) {
	fmt.Printf("Querying packet commitments on SimApp...\n")

	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return nil, err
	}

	queryClient := channeltypesv2.NewQueryClient(clientCtx)
	request := channeltypesv2.QueryPacketCommitmentsRequest{ChannelId: channelID}
	response, err := queryClient.PacketCommitments(context.Background(), &request)
	if err != nil {
		return nil, fmt.Errorf("failed to query packet commitments: %v", err)
	}

	fmt.Printf("Packet commitments: %v, packet height %v\n", response.GetCommitments(), response.GetHeight())
	return response, nil
}

// QueryLightClientLatestHeight queries the ICS07 light client on the EVM
// roll-up for the client state's latest height.
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

// GetStateTransitionProof returns a state transition proof from the Celestia
// prover. The prover will query the Tendermint light client on the EVM roll-up
// for it's last known height and generate a proof from that height all the way
// up to the latest height on SimApp.
func GetStateTransitionProof() (proof []byte, err error) {
	conn, err := grpc.NewClient(celestiaProverEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to the prover service: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	client := proverclient.NewProverClient(conn)
	request := &proverclient.ProveStateTransitionRequest{ClientId: ics07TMContractAddress}

	resp, err := client.ProveStateTransition(ctx, request)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to request state transition proof: %w", err)
	}

	fmt.Printf("Got state transition proof: %x, public values %v\n", resp.GetProof(), resp.GetPublicValues())
	return resp.GetProof(), nil
}

// GetMembershipProof gets a membership proof that the packets in the input are
// present in the state root at the input block height on SimApp.
func GetMembershipProof(input *channeltypesv2.QueryPacketCommitmentsResponse) (proof []byte, err error) {
	conn, err := grpc.NewClient(celestiaProverEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to connect to the prover service: %w", err)
	}
	defer conn.Close()

	client := proverclient.NewProverClient(conn)
	request := &proverclient.ProveStateMembershipRequest{
		Height:   int64(input.GetHeight().RevisionHeight),
		KeyPaths: getKeyPaths(input.Commitments),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	response, err := client.ProveStateMembership(ctx, request)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to request state membership proof: %w", err)
	}

	fmt.Printf("Got membership proof: %x, height %v\n", response.GetProof(), response.GetHeight())
	return response.GetProof(), nil
}

// getKeyPaths returns the Merkle path to packets.
func getKeyPaths(_ []*channeltypesv2.PacketState) []string {
	// TODO: implement
	return []string{}
}
