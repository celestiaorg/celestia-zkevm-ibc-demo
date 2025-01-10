package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibchostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20lib"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/cosmos/solidity-ibc-eureka/abigen/icscore"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/srdtrk/solidity-ibc-eureka/e2e/v8/ethereum"
)

// TODO: fetch these from the `make setup` command output.
const (
	erc20           = "0x94b9b5bd518109db400adc62ab2022d2f0008ff7"
	escrow          = "0x51488819811d51c7a3efcc5f0756740e252da783"
	ibcstore        = "0x686bd6a5be8a2d9d923814b8e9a3957c3c103573"
	ics07Tendermint = "0x25cdbd2bf399341f8fee22ecdb06682ac81fdc37"
	ics20Transfer   = "0xe2c1756b8825c54638f98425c113b51730cc47f6"
	ics26Router     = "0xe53275a1fca119e1c5eeb32e7a72e54835a63936"
	icsCore         = "0x505f890889415cf041001f5190b7800266b0dddd"
)

const (
	// sender is an address on SimApp that will send funds via the MsgTransfer.
	sender = "cosmos1ltvzpwf3eg8e9s7wzleqdmw02lesrdex9jgt0q"
	// receiver is an address on the EVM chain that will receive funds via the MsgTransfer.
	receiver = "0x7f39c581f595b53c5cb19b5a6e5b8f3a0b1f2f6e"
	// denom is the denomination of the token on SimApp.
	denom = "stake"
	// sourceChannel is hard-coded to the name used by the first channel.
	sourceChannel = ibctesting.FirstChannelID
	// sequence is hard-coded to the first sequence number.
	sequence = 1
	// ethereumRPC is the Reth RPC endpoint.
	ethereumRPC = "http://localhost:8545/"
	// ethereumAddress is an address on the EVM chain.
	// ethereumAddress = "0xaF9053bB6c4346381C77C2FeD279B17ABAfCDf4d"
	// ethPrivateKey is the private key for ethereumAddress.
	ethPrivateKey = "0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a"
	// cliendID is for the SP1 Tendermint light client on the EVM roll-up.
	clientID = "07-tendermint-0"
)

func main() {
	err := updateTendermintLightClient()
	if err != nil {
		log.Fatal(err)
	}

	err = receivePacketOnEVM()
	if err != nil {
		log.Fatal(err)
	}
}

// updateTendermintLightClient submits a MsgUpdateClient to the Tendermint light client on the EVM roll-up.
func updateTendermintLightClient() error {
	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	icsCore, err := icscore.NewContract(ethcommon.HexToAddress(icsCore), ethClient)
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
	// sendPacket, err := createSendPacket()
	// if err != nil {
	// 	return err
	// }
	// fmt.Printf("sendPacket %v\n", sendPacket)
	// Update the Tendermint light client on the EVM roll-up with the stateTransitionProof
	// stateTransitionProof := []byte{}
	// TODO: figure out how to encode the state transition proof into this msg
	msg := []byte{}
	tx, err := icsCore.UpgradeClient(getTransactOpts(faucet, eth), clientID, msg)
	if err != nil {
		return err
	}
	receipt := getTxReciept(context.Background(), eth, tx.Hash())
	if ethtypes.ReceiptStatusSuccessful != receipt.Status {
		fmt.Printf("receipt %v\n", receipt)
		fmt.Printf("receipt logs %v\n", receipt.Logs)
		return fmt.Errorf("receipt status want %v, got %v", ethtypes.ReceiptStatusSuccessful, receipt.Status)
	}
	recvBlockNumber := receipt.BlockNumber.Uint64()
	fmt.Printf("recvBlockNumber %v\n", recvBlockNumber)
	return nil
}

func receivePacketOnEVM() error {
	sendPacket, err := createSendPacket()
	if err != nil {
		return err
	}

	packetCommitmentPath := ibchostv2.PacketCommitmentKey(sourceChannel, sequence)
	fmt.Printf("packetCommitmentPath %v\n", packetCommitmentPath)

	packet := ics26router.IICS26RouterMsgsPacket{
		Sequence:         uint32(sendPacket.Sequence),
		SourceChannel:    sendPacket.SourceChannel,
		DestChannel:      sendPacket.DestinationChannel,
		TimeoutTimestamp: sendPacket.TimeoutTimestamp,
		Payloads: []ics26router.IICS26RouterMsgsPayload{
			{
				SourcePort: sendPacket.Payloads[0].SourcePort,
				DestPort:   sendPacket.Payloads[0].DestinationPort,
				Version:    transfertypes.V1,
				Encoding:   transfertypes.EncodingABI,
				Value:      sendPacket.Payloads[0].Value,
			},
		},
	}

	// TODO: replace this with query to celestia-prover after mock circuits
	// are implemented.
	membershipProof := []byte{}

	// TODO: replace this with a real proof height.
	proofHeight := ics26router.IICS02ClientMsgsHeight{
		RevisionNumber: uint32(0),
		RevisionHeight: uint32(10),
	}
	msg := ics26router.IICS26RouterMsgsMsgRecvPacket{
		Packet:          packet,
		ProofCommitment: membershipProof,
		ProofHeight:     proofHeight,
	}

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	ics26Contract, err := ics26router.NewContract(ethcommon.HexToAddress(ics26Router), ethClient)
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
	tx, err := ics26Contract.RecvPacket(getTransactOpts(faucet, eth), msg)
	if err != nil {
		return err
	}

	receipt := getTxReciept(context.Background(), eth, tx.Hash())
	if ethtypes.ReceiptStatusSuccessful != receipt.Status {
		return fmt.Errorf("receipt status want %v, got %v", ethtypes.ReceiptStatusSuccessful, receipt.Status)
	}
	recvBlockNumber := receipt.BlockNumber.Uint64()
	fmt.Printf("recvBlockNumber %v\n", recvBlockNumber)
	return nil
}

// TODO: refactor this to de-duplicate code from createMsgSendPacket
func createSendPacket() (channeltypesv2.Packet, error) {
	coin := sdktypes.NewCoin(denom, math.NewInt(100))
	transferPayload := ics20lib.ICS20LibFungibleTokenPacketData{
		Denom:    coin.Denom,
		Amount:   coin.Amount.BigInt(),
		Sender:   sender,
		Receiver: receiver,
		Memo:     "test transfer",
	}
	transferBz, err := ics20lib.EncodeFungibleTokenPacketData(transferPayload)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}
	payload := channeltypesv2.Payload{
		SourcePort:      transfertypes.PortID,
		DestinationPort: transfertypes.PortID,
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingABI,
		Value:           transferBz,
	}

	return channeltypesv2.Packet{
		Sequence:           sequence,
		SourceChannel:      ibctesting.FirstChannelID,
		DestinationChannel: ibctesting.FirstClientID,
		TimeoutTimestamp:   uint64(time.Now().Add(30 * time.Minute).Unix()),
		Payloads:           []channeltypesv2.Payload{payload},
	}, nil
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
