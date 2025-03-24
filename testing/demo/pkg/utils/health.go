package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// CheckSimappNodeHealth verifies that the SimApp node is up and running.
func CheckSimappNodeHealth(nodeURI string, maxRetries int) error {
	var lastErr error
	backoffDuration := time.Second

	for i := 0; i < maxRetries; i++ {
		// Create a client with a timeout
		httpClient := &http.Client{
			Timeout: time.Second * 5,
		}

		// Make a request to the /status endpoint
		resp, err := httpClient.Get(fmt.Sprintf("%s/status", nodeURI))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}

		if err != nil {
			lastErr = err
			fmt.Printf("Node health check failed: %v. Retrying in %v...\n", err, backoffDuration)
		} else {
			resp.Body.Close()
			lastErr = fmt.Errorf("node returned status code %d", resp.StatusCode)
			fmt.Printf("Node health check failed: %v. Retrying in %v...\n", lastErr, backoffDuration)
		}

		time.Sleep(backoffDuration)
		backoffDuration *= 2 // Exponential backoff
	}

	return fmt.Errorf("node health check failed after %d attempts: %v", maxRetries, lastErr)
}

func CheckEthereumNodeHealth(ethereumRPC string) error {
	// Check if Ethereum node is healthy
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum client: %v", err)
	}

	// Try to get the latest block to verify the node is working
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = ethClient.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("ethereum node is not responding correctly: %v", err)
	}

	return nil
}
