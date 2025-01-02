package main

import "fmt"

func main() {
	fmt.Printf("Hello from relayer\n")
	// Update the tendermint client on the reth rollup (querying the prover process for the groth16 proof proving the simapps state transition from a previous trusted height to a new one)
	// Need to query SimApp's IBC module to learn if there are any new packets to relay.
	// TODO: can we use an existing relayer for this functionality?
}
