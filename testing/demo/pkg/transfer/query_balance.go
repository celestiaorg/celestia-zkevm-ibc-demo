package main

import (
	"fmt"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ibcerc20"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func queryBalance() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return fmt.Errorf("failed to extract deployed contract addresses: %w", err)
	}
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ibcERC20, err := ibcerc20.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return fmt.Errorf("failed to create IBCERC20 contract: %w", err)
	}

	actualDenom, err := ibcERC20.Name(getCallOpts())
	if err != nil {
		return fmt.Errorf("failed to get denom on Ethereum: %w", err)
	}

	// Recreate the full denom path
	denomOnEthereum := transfertypes.NewDenom(denom, transfertypes.NewHop(transfertypes.PortID, tendermintClientID))
	if actualDenom != denomOnEthereum.Path() {
		return fmt.Errorf("denom on Ethereum does not match expected denom: %s != %s", actualDenom, denomOnEthereum.Path())
	}

	actualSymbol, err := ibcERC20.Symbol(nil)
	if err != nil {
		return fmt.Errorf("failed to get symbol on Ethereum: %w", err)
	}

	if actualSymbol != denomOnEthereum.Path() {
		return fmt.Errorf("symbol on Ethereum does not match expected symbol: %s != %s", actualSymbol, denomOnEthereum.Path())
	}

	actualFullDenom, err := ibcERC20.FullDenomPath(nil)
	if err != nil {
		return fmt.Errorf("failed to get full denom path on Ethereum: %w", err)
	}

	if denomOnEthereum.Path() != actualFullDenom {
		return fmt.Errorf("full denom on Ethereum does not match expected full denom: %s != %s", actualFullDenom, denomOnEthereum.Path())
	}

	// User balance on Ethereum
	userBalance, err := ibcERC20.BalanceOf(nil, ethcommon.HexToAddress(receiver))
	if err != nil {
		return fmt.Errorf("failed to get user balance on Ethereum: %w", err)
	}

	if userBalance.Int64() != transferAmount.Int64() {
		return fmt.Errorf("user balance on Ethereum does not match expected balance: %v != %v", userBalance.Int64(), transferAmount.Int64())
	}

	// ICS20 contract balance on Ethereum
	ics20TransferBalance, err := ibcERC20.BalanceOf(nil, ethcommon.HexToAddress(addresses.ICS20Transfer))
	if err != nil {
		return fmt.Errorf("failed to get ICS20 contract balance on Ethereum: %w", err)
	}
	if ics20TransferBalance.Int64() != 0 {
		return fmt.Errorf("ICS20 contract balance on Ethereum is not zero: %v", ics20TransferBalance.Int64())
	}

	fmt.Printf("User balance on Ethereum: %v\n", userBalance.Int64())
	return nil
}
