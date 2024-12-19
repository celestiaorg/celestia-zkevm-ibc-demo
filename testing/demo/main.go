package main

import (
	"context"
	"fmt"
	"os"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

// TODO: we can delete this whole file because it appears unused.
// See: https://github.com/celestiaorg/celestia-zkevm-ibc-demo/issues/49
var rootCmd = &cobra.Command{
	Use:   "demo",
	Short: "Runs the full ZK IBC demo",
	Long: `Runs the full ZK IBC demo, using docker-compose to run a ZK enabled Cosmos chain, a Reth Rollup
connected to Celestia. It deploys the IBC smart contracts, establishes a connection between the rollup and chain,
and performs a token transfer`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if err := pkg.Start(cmd.Context()); err != nil {
			return err
		}
		defer func() {
			stopErr := pkg.Stop(context.Background())
			if stopErr != nil {
				err = multierror.Append(err, stopErr)
			}
		}()
		if err := pkg.DeployContracts(); err != nil {
			return err
		}
		if err := pkg.EstablishIBCConnection(); err != nil {
			return err
		}
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the demo environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.Start(cmd.Context())
	},
}

var setupIBCContractsCmd = &cobra.Command{
	Use:   "setup-ibc-contracts",
	Short: "Setup IBC contracts",
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.DeployContracts()
	},
}

var createIBCConnectionCmd = &cobra.Command{
	Use:   "create-ibc-connection",
	Short: "Create IBC connection between chains",
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.EstablishIBCConnection()
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops and cleans up the demo environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.Stop(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(setupIBCContractsCmd)
	rootCmd.AddCommand(createIBCConnectionCmd)
	rootCmd.AddCommand(stopCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
