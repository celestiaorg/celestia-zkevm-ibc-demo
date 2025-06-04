package main

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ibcerc20"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20transfer"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func queryBalances() error {
	err := queryBalanceOnSimApp()
	if err != nil {
		return fmt.Errorf("failed to query balance on SimApp: %w", err)
	}
	err = queryBalanceOnEthereum()
	if err != nil {
		return fmt.Errorf("failed to query balance on Ethereum: %w", err)
	}
	return nil
}

func queryBalanceOnSimApp() error {
	currentBalance, err := getSimappUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for SimApp: %w", err)
	}
	fmt.Printf("Current balance on SimApp: %v\n", currentBalance.Int64())

	return nil
}

func queryBalanceOnEthereum() error {
	userBalance, err := getEvmUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for Ethereum: %w", err)
	}
	fmt.Printf("Current balance on Ethereum: %v\n", userBalance.Int64())

	return nil
}

func getSimappUserBalance() (math.Int, error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to setup client context: %w", err)
	}

	senderAcc, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to convert sender address: %w", err)
	}

	queryClient := banktypes.NewQueryClient(clientCtx.GRPCClient)
	resp, err := queryClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: senderAcc.String(),
		Denom:   denom,
	})
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to query balance on SimApp: %w", err)
	}

	return resp.Balance.Amount, nil
}

func getEvmUserBalance() (math.Int, error) {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to extract deployed contract addresses: %w", err)
	}
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ics20Transfer, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to create ICS20Transfer contract: %w", err)
	}

	denomOnEthereum := transfertypes.NewDenom(denom, transfertypes.NewHop(transfertypes.PortID, tendermintClientID))
	ibcERC20Address, _ := ics20Transfer.IbcERC20Contract(nil, denomOnEthereum.Path())
	if ibcERC20Address == (ethcommon.Address{}) {
		fmt.Printf("IBCErc20 contract has not been deployed for the specified denom: %s\n", denomOnEthereum.Path())
		return math.NewInt(0), nil
	}

	ibcERC20, err := ibcerc20.NewContract(ibcERC20Address, ethClient)
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to create IBCERC20 contract: %w", err)
	}

	actualDenom, err := ibcERC20.Name(getCallOpts())
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to get denom on Ethereum: %w", err)
	}

	if actualDenom != denomOnEthereum.Path() {
		return math.NewInt(0), fmt.Errorf("denom on Ethereum does not match expected denom: %s != %s", actualDenom, denomOnEthereum.Path())
	}

	actualSymbol, err := ibcERC20.Symbol(nil)
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to get symbol on Ethereum: %w", err)
	}

	if actualSymbol != denomOnEthereum.Path() {
		return math.NewInt(0), fmt.Errorf("symbol on Ethereum does not match expected symbol: %s != %s", actualSymbol, denomOnEthereum.Path())
	}

	actualFullDenom, err := ibcERC20.FullDenomPath(nil)
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to get full denom path on Ethereum: %w", err)
	}

	if denomOnEthereum.Path() != actualFullDenom {
		return math.NewInt(0), fmt.Errorf("full denom on Ethereum does not match expected full denom: %s != %s", actualFullDenom, denomOnEthereum.Path())
	}

	userBalance, err := ibcERC20.BalanceOf(nil, ethcommon.HexToAddress(receiver))
	if err != nil {
		return math.NewInt(0), fmt.Errorf("failed to get user balance on Ethereum: %w", err)
	}

	return math.NewInt(userBalance.Int64()), nil
}
