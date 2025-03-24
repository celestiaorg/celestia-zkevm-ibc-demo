package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getProverSTFKey() (key [32]byte, err error) {
	info, err := getCelestiaProverInfo()
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get celestia-prover info %w", err)
	}

	decoded, err := hex.DecodeString(strings.TrimPrefix(info.StateTransitionVerifierKey, "0x"))
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to decode state transition verifier key %w", err)
	}

	copy(key[:], decoded)
	return key, nil
}

func getProverMembershipKey() (key [32]byte, err error) {
	info, err := getCelestiaProverInfo()
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get celestia-prover info %w", err)
	}

	decoded, err := hex.DecodeString(strings.TrimPrefix(info.StateMembershipVerifierKey, "0x"))
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to decode membership verifier key %w", err)
	}

	copy(key[:], decoded)
	return key, nil
}

func getCelestiaProverInfo() (*proverclient.InfoResponse, error) {
	celestiaProverConn, err := grpc.NewClient(celestiaProverRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer celestiaProverConn.Close()

	proverClient := proverclient.NewProverClient(celestiaProverConn)
	info, err := proverClient.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get celestia-prover info %w", err)
	}
	return info, nil
}
