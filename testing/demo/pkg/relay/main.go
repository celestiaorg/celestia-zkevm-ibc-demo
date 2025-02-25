package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	transfer "github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/transfer"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics02client"
	"github.com/cosmos/solidity-ibc-eureka/abigen/sp1ics07tendermint"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/ethereum"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// ethereumRPC is the Reth RPC endpoint.
	ethereumRPC = "http://localhost:8545/"
	// ethereumAddress is an address on the EVM chain.
	// ethereumAddress = "0xaF9053bB6c4346381C77C2FeD279B17ABAfCDf4d"
	// ethPrivateKey is the private key for ethereumAddress.
	ethPrivateKey = "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"
	// cliendID is for the SP1 Tendermint light client on the EVM roll-up.
	clientID = "07-tendermint-0"
	// sender is an address on SimApp that will send funds via the MsgTransfer.
	Sender = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"
	// receiver is an address on the EVM chain that will receive funds via the MsgTransfer.
	Receiver = "0x7f39c581f595b53c5cb19b5a6e5b8f3a0b1f2f6e"
	// denom is the denomination of the token on SimApp.
	Denom  = "stake"
	Amount = 100
	// SenderInitialBalance is the initial balance of the sender from genesis.
	SenderInitialBalance = 274883996352
	// ReceiverInitialBalance is the initial balance of the receiver.
	ReceiverInitialBalance = 0
)

func main() {
	err := updateTendermintLightClient()
	if err != nil {
		log.Fatal(err)
	}
}

// updateTendermintLightClient submits a MsgUpdateClient to the Tendermint light client on the EVM roll-up.
func updateTendermintLightClient() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Extracted deployed contract addresses: %#v\n", addresses)

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	icsClient, err := ics02client.NewContract(ethcommon.HexToAddress(addresses.ICS02Client), ethClient)
	if err != nil {
		return err
	}
	faucet, err := crypto.ToECDSA(ethcommon.FromHex(ethPrivateKey))
	if err != nil {
		return err
	}
	eth, err := ethereum.NewEthereum(context.Background(), ethereumRPC, nil, faucet)
	if err != nil {
		return err
	}

	// Connect to the Celestia prover
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Getting celestia prover info...\n")
	proverClient := proverclient.NewProverClient(conn)
	info, err := proverClient.Info(context.Background(), &proverclient.InfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to get celestia prover info %w", err)
	}
	fmt.Printf("Got celestia prover info with StateTransitionVerifierKey: %v\n", info.StateTransitionVerifierKey)
	verifierKeyDecoded, err := hex.DecodeString(strings.TrimPrefix(info.StateTransitionVerifierKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode verifier key %w", err)
	}
	var verifierKey [32]byte
	copy(verifierKey[:], verifierKeyDecoded)

	request := &proverclient.ProveStateTransitionRequest{ClientId: addresses.ICS07Tendermint}
	// Get state transition proof from Celestia prover with retry logic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var resp *proverclient.ProveStateTransitionResponse
	for retries := 0; retries < 3; retries++ {
		resp, err = proverClient.ProveStateTransition(ctx, request)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled while getting state transition proof: %w", ctx.Err())
		}
		time.Sleep(time.Second * time.Duration(retries+1))
	}
	if err != nil {
		return fmt.Errorf("failed to get state transition proof after retries: %w", err)
	}
	arguments, err := getUpdateClientArguments()
	if err != nil {
		return err
	}
	encoded, err := arguments.Pack(sp1ics07tendermint.IUpdateClientMsgsMsgUpdateClient{
		Sp1Proof: sp1ics07tendermint.ISP1MsgsSP1Proof{
			VKey:         verifierKey,
			PublicValues: resp.PublicValues,
			Proof:        resp.Proof,
		},
	})
	if err != nil {
		return fmt.Errorf("error packing msg %w", err)
	}

	fmt.Printf("Invoking icsCore.UpdateClient...\n")
	tx, err := icsClient.UpdateClient(getTransactOpts(faucet, eth), clientID, encoded)
	if err != nil {
		return err
	}
	receipt := getTxReciept(context.Background(), eth, tx.Hash())
	if ethtypes.ReceiptStatusSuccessful != receipt.Status {
		fmt.Printf("receipt %v and logs %v\n", receipt, receipt.Logs)
		return fmt.Errorf("receipt status want %v, got %v", ethtypes.ReceiptStatusSuccessful, receipt.Status)
	}
	recvBlockNumber := receipt.BlockNumber.Uint64()
	fmt.Printf("recvBlockNumber %v\n", recvBlockNumber)
	return nil
}

// ackMembershipOnRethAndUpdatedBalances queries the Reth node for the membership proof of ack, submits it to SimApp
// and makes sure balances are updated on both chains.
func ackMembershipOnRethAndUpdatedBalances() error {
	// Query the Membership proof of ack on the Reth node
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Extracted deployed contract addresses: %#v\n", addresses)

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}

	// TODO: Replace this with Ack key
	key := ethcommon.HexToHash("0x123...abc")

	// Prepare the arguments
	// Storage proof takes the address and the storage slot as arguments; here, only the key is shown for simplicity
	args := map[string]interface{}{
		"address":     "0xAddress", // What is this address going to be with ack?
		"key":         key.Hex(),
		"blockNumber": "latest", // or provide a specific block number
	}

	proof := ethcommon.Hash{}
	proofHeight := big.NewInt(0)
	err = ethClient.Client().CallContext(context.Background(), &proof, "eth_getProof", args["address"], []string{args["key"].(string)}, proofHeight)
	if err != nil {
		return err
	}

	// TODO: Parse Ack from Ethereum events

	// Embed it in the ack packet that will be submitted to the SimApp chain
	// Q: should this be the relayer?
	ackMsg := channeltypesv2.NewMsgAcknowledgement(packet, ack, proof.Bytes(), proofHeight, Sender)

	txHash, err := submitMessageAck(ackMsg)
	if err != nil {
		return err
	}

	// Query the updated balance from the Reth node (increased)
	receiverBalance, err := ethClient.BalanceAt(context.Background(), ethcommon.HexToAddress(Receiver), nil)
	if err != nil {
		return err
	}
	if receiverBalance != big.Int(ReceiverInitialBalance)+Amount {
		return fmt.Errorf("receiver balance not updated")

	}

	// Query the updated balance from the SimApp chain (decreased)
	senderBalance, err := utils.GetAccountBalance(Sender)
	if err != nil {
		return err
	}

	return nil
}

func submitMessageAck(msg *channeltypesv2.MsgAcknowledgement) (txHash string, err error) {
	clientCtx, err := utils.SetupClientContext()
	if err != nil {
		return "", fmt.Errorf("failed to setup client context: %v", err)
	}

	fmt.Printf("Broadcasting MsgTransfer...\n")
	response, err := utils.BroadcastMessages(clientCtx, Sender, 200_000, msg)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast MsgTransfer %w", err)
	}

	if response.Code != 0 {
		return "", fmt.Errorf("failed to execute MsgTransfer %v", response.RawLog)
	}
	fmt.Printf("Broadcasted MsgTransfer. Response code: %v, tx hash: %v\n", response.Code, response.TxHash)
	return response.TxHash, nil
}

func getTransactOpts(key *ecdsa.PrivateKey, chain ethereum.Ethereum) *bind.TransactOpts {
	ethClient, err := ethclient.Dial(chain.RPC)
	if err != nil {
		log.Fatal(err)
	}

	fromAddress := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		nonce = 0
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(key, chain.ChainID)
	if err != nil {
		log.Fatal(err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.GasPrice = gasPrice

	// Set a specific gas limit
	txOpts.GasLimit = 3000000 // Example gas limit; adjust as needed

	return txOpts
}

func getTxReciept(ctx context.Context, chain ethereum.Ethereum, hash ethcommon.Hash) *ethtypes.Receipt {
	ethClient, err := ethclient.Dial(chain.RPC)
	if err != nil {
		log.Fatal(err)
	}

	var receipt *ethtypes.Receipt
	err = utils.WaitForCondition(time.Second*30, time.Second, func() (bool, error) {
		receipt, err = ethClient.TransactionReceipt(ctx, hash)
		if err != nil {
			return false, nil
		}

		return receipt != nil, nil
	})
	if err != nil {
		log.Fatalf("Failed to fetch receipt: %v", err)
	}

	// Log more details about the receipt
	fmt.Printf("Transaction hash: %s\n", hash.Hex())
	fmt.Printf("Block number: %d\n", receipt.BlockNumber.Uint64())
	fmt.Printf("Gas used: %d\n", receipt.GasUsed)
	fmt.Printf("Logs: %v\n", receipt.Logs)
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		fmt.Println("Transaction failed. Inspect logs or contract.")
	}

	return receipt
}

func getUpdateClientArguments() (abi.Arguments, error) {
	var updateClientABI = "[{\"type\":\"function\",\"name\":\"updateClient\",\"stateMutability\":\"pure\",\"inputs\":[{\"name\":\"o3\",\"type\":\"tuple\",\"internalType\":\"struct IUpdateClientMsgs.MsgUpdateClient\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"struct ISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]}],\"outputs\":[]}]"

	parsed, err := abi.JSON(strings.NewReader(updateClientABI))
	if err != nil {
		return nil, err
	}

	return parsed.Methods["updateClient"].Inputs, nil
}
