package main

import (
	"context"
	"fmt"
	"log"
	"time"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibchostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// celestiaProverEndpoint is the endpoint of the Celestia prover.
	celestiaProverEndpoint = "localhost:50051"
	// channelID is the channel ID on SimApp.
	// TODO: fetch this from the `make setup` command output.
	channelID = "channel-0"
	// ics07TMContractAddress is the contract address of the ICS07 light client on the EVM roll-up.
	// TODO: fetch this from the `make setup` command output.
	ics07TMContractAddress = "0x25cdbd2bf399341f8fee22ecdb06682ac81fdc37"
	// sourceChannel is hard-coded to the name used by the first channel.
	sourceChannel = ibctesting.FirstChannelID
	// sequence is hard-coded to the first sequence number used by the MsgSendPacket.
	sequence = 1
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
	// MsgRecvPacket to the EVM rollup. See solidity-ibc-eureka for example.
	//
	// https://github.com/cosmos/solidity-ibc-eureka/blob/febaabb6915eccfd3e1922793bc0936cd0b4fdfb/e2e/interchaintestv8/ibc_eureka_test.go#L816
	packetCommitmentPath := ibchostv2.PacketCommitmentKey(sourceChannel, sequence)
	fmt.Printf("packetCommitmentPath %v\n", packetCommitmentPath)

	// Note: solidity-ibc-eureka tests wrap the MsgSendPacket that we have in transfer.
	//
	// https://github.com/cosmos/solidity-ibc-eureka/blob/febaabb6915eccfd3e1922793bc0936cd0b4fdfb/e2e/interchaintestv8/ibc_eureka_test.go#L779-L787

	// proofHeight, ucAndMemProof := updateClientAndMembershipProof(ctx, simd, pt, [][]byte{packetCommitmentPath})
	// packet := ics26router.IICS26RouterMsgsPacket{
	// 	Sequence:         uint32(sendPacket.Sequence),
	// 	SourceChannel:    sendPacket.SourceChannel,
	// 	DestChannel:      sendPacket.DestinationChannel,
	// 	TimeoutTimestamp: sendPacket.TimeoutTimestamp,
	// 	Payloads: []ics26router.IICS26RouterMsgsPayload{
	// 		{
	// 			SourcePort: sendPacket.Payloads[0].SourcePort,
	// 			DestPort:   sendPacket.Payloads[0].DestinationPort,
	// 			Version:    transfertypes.V1,
	// 			Encoding:   transfertypes.EncodingABI,
	// 			Value:      sendPacket.Payloads[0].Value,
	// 		},
	// 	},
	// }
	// msg := ics26router.IICS26RouterMsgsMsgRecvPacket{
	// 	Packet:          packet,
	// 	ProofCommitment: ucAndMemProof,
	// 	ProofHeight:     *proofHeight,
	// }
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
	// Are packet commitments the correct data type to be proving here?
	// TODO: investigate existing IBC relayer implementations.
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

// getKeyPaths returns a list of strings where each string is a Merkle path for
// a leaf to the state root.
func getKeyPaths(_ []*channeltypesv2.PacketState) []string {
	// TODO: implement
	return []string{}
}
