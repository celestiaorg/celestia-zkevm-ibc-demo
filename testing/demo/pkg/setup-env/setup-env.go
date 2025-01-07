package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// Define the source and destination files
	srcFile := filepath.Join(wd, "solidity-ibc-eureka/.env.example")
	destFile := filepath.Join(wd, "solidity-ibc-eureka/.env")

	// Define the environment variables to replace
	envVars := map[string]string{
		"RPC_URL":            "http://localhost:8545",
		"TENDERMINT_RPC_URL": "http://localhost:5123",
		"PRIVATE_KEY":        "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a",
		"CONTRACT_ADDRESS":   "0x2854CFaC53FCaB6C95E28de8C91B96a31f0af8DD",
		"E2E_FAUCET_ADDRESS": "0xaF9053bB6c4346381C77C2FeD279B17ABAfCDf4d",
	}

	// Copy the source file to the destination file
	if err := copyFile(srcFile, destFile); err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		return
	}

	// Replace the environment variables in the destination file
	if err := replaceEnvVars(destFile, envVars); err != nil {
		fmt.Printf("Error replacing environment variables: %v\n", err)
		return
	}

	fmt.Println("Environment variables replaced successfully.")
}

// copyFile copies the contents of the source file to the destination file
func copyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(dest, input, 0644); err != nil {
		return err
	}

	return nil
}

// replaceEnvVars replaces the specified environment variables in the file
func replaceEnvVars(filePath string, envVars map[string]string) error {
	fmt.Printf("envVars %v\n", envVars)
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		for key, value := range envVars {
			if strings.HasPrefix(line, key+"=") {
				lines[i] = fmt.Sprintf("%s=%s", key, value)
			}
		}
	}

	// Add any new environment variables that were not in the original file
	for key, value := range envVars {
		found := false
		for _, line := range lines {
			if strings.HasPrefix(line, key+"=") {
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return err
	}

	return nil
}
