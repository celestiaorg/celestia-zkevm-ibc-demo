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
	err = InitializeTendermintLightClientOnEVMRollup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = RegisterCounterpartyOnSimapp()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
