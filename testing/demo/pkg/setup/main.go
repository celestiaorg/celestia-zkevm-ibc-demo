package main

import (
	"fmt"
	"os"
)

func main() {
	err := InitializeGroth16LightClientOnSimapp()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = SetupSp1TendermintClientAndChannelsOnReth()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}