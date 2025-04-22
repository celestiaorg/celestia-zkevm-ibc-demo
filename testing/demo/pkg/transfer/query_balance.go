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

var previousBalanceOnSimapp math.Int
func queryBalance() error {
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

	// If the balance is increased, we're transferring from EVM to SimApp
	if currentBalance.GT(previousBalanceOnSimapp) {
		netTransferAmount := currentBalance.Sub(previousBalanceOnSimapp)
		fmt.Printf("currentBalance: %v\n", currentBalance)
		expectedBalance := previousBalanceOnSimapp.Add(netTransferAmount)
		fmt.Printf("expectedBalance: %v\n", expectedBalance)
		if !currentBalance.Equal(expectedBalance) {
			return fmt.Errorf("sender balance on SimApp not match expected balance: %v != %v", currentBalance.Int64(), expectedBalance.Int64())
		}
		fmt.Printf("Initial balance on Simapp: %v\nCurrent balance on SimApp: %v\nBalance increase on SimApp (transfer amount + gas fees): %v\n", previousBalanceOnSimapp, currentBalance.Int64(), currentBalance.Sub(previousBalanceOnSimapp).Int64())
		// otherwise, we're transferring from SimApp to EVM
	} else if currentBalance.LT(previousBalanceOnSimapp) {
		expectedBalance := previousBalanceOnSimapp.Sub(currentBalance)
		if currentBalance.LT(expectedBalance) {
			return fmt.Errorf("sender balance on SimApp not match expected balance: %v != %v", currentBalance.Int64(), expectedBalance.Int64())
		}
		fmt.Printf("Initial balance on SimApp: %v\nCurrent balance on SimApp: %v\nDifference on SimApp (transfer amount + gas fees): %v\n", previousBalanceOnSimapp, currentBalance.Int64(), previousBalanceOnSimapp.Sub(currentBalance).Int64())
	}

	return nil
}

var previousBalanceOnEvmChain math.Int
func queryBalanceOnEthereum() error {
	userBalance, ibcERC20, ICS20TransferAddress, err := getEvmUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for Ethereum: %w", err)
	}

	fmt.Printf("receiver: %s\n", receiver)
	fmt.Printf("user balance: %v\n", userBalance.Int64())

	// In balance increased, we're transferring from SimApp to Ethereum
	if userBalance.Int64() > previousBalanceOnEvmChain.Int64() {

		expectecBalance := previousBalanceOnEvmChain.Add(transferAmount)
		if userBalance.Int64() != expectecBalance.Int64() {
			fmt.Printf("user balance on Ethereum does not match expected balance: %v != %v", userBalance.Int64(), expectecBalance.Int64())
		}

		fmt.Printf("Initial balance on Ethereum: %v\nCurrent balance on Ethereum: %v\nBalance increased on Ethereum by: %v\n", previousBalanceOnEvmChain, userBalance.Int64(), expectecBalance.Sub(previousBalanceOnEvmChain).Int64())
	} else if userBalance.Int64() < previousBalanceOnEvmChain.Int64() {

		expectecBalance := previousBalanceOnEvmChain.Sub(math.NewInt(transferBackAmount.Int64()))
		fmt.Printf("expectecBalance: %v\n", expectecBalance)
		if userBalance.Int64() != expectecBalance.Int64() {
			fmt.Printf("user balance on Ethereum does not match expected balance: %v != %v", userBalance.Int64(), expectecBalance.Int64())
		}

		fmt.Printf("Initial balance on Ethereum: %v\nCurrent balance on Ethereum: %v\nBalance decrease on Ethereum: %v\n", previousBalanceOnEvmChain, userBalance.Int64(), userBalance.Sub(previousBalanceOnEvmChain).Int64())
	}

	ics20TransferBalance, err := ibcERC20.BalanceOf(nil, ethcommon.HexToAddress(ICS20TransferAddress))
	if err != nil {
		return fmt.Errorf("failed to get ICS20 contract balance on Ethereum: %w", err)
	}
	if ics20TransferBalance.Int64() != 0 {
		return fmt.Errorf("ICS20 contract balance on Ethereum is not zero: %v", ics20TransferBalance.Int64())
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
	ibcERC20Address, err := ics20Transfer.IbcERC20Contract(nil, denomOnEthereum.Path())
	if err != nil {
		return math.NewInt(0), nil, "", fmt.Errorf("failed to get IBCERC20 contract address: %w", err)
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
	previousBalanceOnSimapp, err = getSimappUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for SimApp: %w", err)
	}
	previousBalanceOnEvmChain, _, _, err = getEvmUserBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance for EVM: %w", err)
	}
	return nil
}
