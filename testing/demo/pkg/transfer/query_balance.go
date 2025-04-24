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

func queryAndAssertBalances() error {
	err := assertBalanceOnSimApp()
	if err != nil {
		return fmt.Errorf("failed to assert balance on SimApp: %w", err)
	}
	err = assertBalanceOnEthereum()
	if err != nil {
		return fmt.Errorf("failed to assert balance on Ethereum: %w", err)
	}
	return nil
}

func assertBalanceOnSimApp() error {
	currentBalance, err := getSimappUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for SimApp: %w", err)
	}

	difference := currentBalance.Sub(initialBalanceOnSimapp)

	switch {
	case difference.IsPositive():
		// Received tokens from EVM
		netTransferAmount := currentBalance.Sub(initialBalanceOnSimapp)
		expectedBalance := initialBalanceOnSimapp.Add(netTransferAmount)
		if !currentBalance.Equal(expectedBalance) {
			return fmt.Errorf("sender balance on SimApp does not match expected balance: %v != %v\n", currentBalance, expectedBalance)
		}
		fmt.Printf("Initial balance on SimApp: %v\n", initialBalanceOnSimapp)
		fmt.Printf("Current balance on SimApp: %v\n", currentBalance.Int64())
		fmt.Printf("Received %v tokens after the gas fees from EVM\n", difference)

	case difference.IsNegative():
		// Sent tokens to EVM
		expectedBalance := initialBalanceOnSimapp.Sub(currentBalance)
		if currentBalance.LT(expectedBalance) {
			return fmt.Errorf("sender balance on SimApp does not match expected balance: %v != %v\n", currentBalance, expectedBalance)
		}
		fmt.Printf("Initial balance on SimApp: %v\n", initialBalanceOnSimapp)
		fmt.Printf("Current balance on SimApp: %v\n", currentBalance)
		fmt.Printf("Sent %v tokens including gas fees to EVM chain\n", difference.Neg())

	default:
		fmt.Printf("Balance unchanged: %v\n", currentBalance)
	}

	return nil
}

func assertBalanceOnEthereum() error {
	userBalance, ibcERC20, ICS20TransferAddress, err := getEvmUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for Ethereum: %w", err)
	}

	difference := userBalance.Sub(initialBalanceOnEvm)

	switch {
	case difference.IsPositive():
		// Received tokens from SimApp
		expectecBalance := initialBalanceOnEvm.Add(transferAmount)
		if userBalance.Int64() != expectecBalance.Int64() {
			fmt.Printf("user balance on Ethereum does not match expected balance: %v != %v\n", userBalance, expectecBalance)
		}
		fmt.Printf("Initial balance on EVM chain: %v\n", initialBalanceOnEvm)
		fmt.Printf("Current balance on EVM chain: %v\n", userBalance)
		fmt.Printf("Received %v tokens from SimApp\n", difference)
	case difference.IsNegative():
		// Sent tokens to SimApp
		expectecBalance := initialBalanceOnEvm.Sub(math.NewInt(transferBackAmount.Int64()))
		if userBalance.Int64() != expectecBalance.Int64() {
			fmt.Printf("user balance on Ethereum does not match expected balance: %v != %v\n", userBalance, expectecBalance)
		}

		fmt.Printf("Initial balance on EVM chain: %v\n", initialBalanceOnEvm)
		fmt.Printf("Current balance on EVM chain: %v\n", userBalance)
		fmt.Printf("Sent %v tokens to SimApp chain\n", difference.Neg())

	default:
		fmt.Printf("Balance unchanged: %v\n", userBalance)
	}

	ics20TransferBalance, err := ibcERC20.BalanceOf(nil, ethcommon.HexToAddress(ICS20TransferAddress))
	if err != nil {
		return fmt.Errorf("failed to get ICS20 contract balance on Ethereum: %w", err)
	}
	if ics20TransferBalance.Int64() != 0 {
		return fmt.Errorf("ICS20 contract balance on Ethereum is not zero: %v\n", ics20TransferBalance.Int64())
	}

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

func getEvmUserBalance() (math.Int, *ibcerc20.Contract, string, error) {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to extract deployed contract addresses: %w", err)
	}
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	ics20Transfer, err := ics20transfer.NewContract(ethcommon.HexToAddress(addresses.ICS20Transfer), ethClient)
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to create ICS20Transfer contract: %w", err)
	}

	denomOnEthereum := transfertypes.NewDenom(denom, transfertypes.NewHop(transfertypes.PortID, tendermintClientID))
	ibcERC20Address, _ := ics20Transfer.IbcERC20Contract(nil, denomOnEthereum.Path())
	if ibcERC20Address == (ethcommon.Address{}) {
		fmt.Printf("IBCErc20 contract has not been deployed for the specified denom: %s\n", denomOnEthereum.Path())
		return math.NewInt(0), nil, "", nil
	}

	ibcERC20, err := ibcerc20.NewContract(ibcERC20Address, ethClient)
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to create IBCERC20 contract: %w", err)
	}

	actualDenom, err := ibcERC20.Name(getCallOpts())
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to get denom on Ethereum: %w", err)
	}

	if actualDenom != denomOnEthereum.Path() {
		return math.NewInt(0), nil, "", fmt.Errorf("denom on Ethereum does not match expected denom: %s != %s", actualDenom, denomOnEthereum.Path())
	}

	actualSymbol, err := ibcERC20.Symbol(nil)
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to get symbol on Ethereum: %w", err)
	}

	if actualSymbol != denomOnEthereum.Path() {
		return math.NewInt(0), nil, "", fmt.Errorf("symbol on Ethereum does not match expected symbol: %s != %s", actualSymbol, denomOnEthereum.Path())
	}

	actualFullDenom, err := ibcERC20.FullDenomPath(nil)
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to get full denom path on Ethereum: %w", err)
	}

	if denomOnEthereum.Path() != actualFullDenom {
		return math.NewInt(0), nil, "", fmt.Errorf("full denom on Ethereum does not match expected full denom: %s != %s", actualFullDenom, denomOnEthereum.Path())
	}

	userBalance, err := ibcERC20.BalanceOf(nil, ethcommon.HexToAddress(receiver))
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to get user balance on Ethereum: %w", err)
	}

	return math.NewInt(userBalance.Int64()), ibcERC20, addresses.ICS20Transfer, nil
}

func updateBalances() error {
	var err error
	initialBalanceOnSimapp, err = getSimappUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for SimApp: %w", err)
	}
	initialBalanceOnEvm, _, _, err = getEvmUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for EVM: %w", err)
	}
	return nil
}
