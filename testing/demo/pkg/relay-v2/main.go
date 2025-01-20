package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"cosmossdk.io/math"
	proverclient "github.com/celestiaorg/celestia-zkevm-ibc-demo/provers/client"
	"github.com/celestiaorg/celestia-zkevm-ibc-demo/testing/demo/pkg/utils"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibchostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20lib"
	"github.com/cosmos/solidity-ibc-eureka/abigen/ics26router"
	"github.com/cosmos/solidity-ibc-eureka/abigen/icscore"
	"github.com/cosmos/solidity-ibc-eureka/abigen/sp1ics07tendermint"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/srdtrk/solidity-ibc-eureka/e2e/v8/ethereum"
	"google.golang.org/grpc"
)

// TODO: fetch these from the `make setup` command output.
// const (
// 	ics26Router            = "0xe53275a1fca119e1c5eeb32e7a72e54835a63936"
// 	icsCore                = "0x505f890889415cf041001f5190b7800266b0dddd"
// 	ics07TMContractAddress = "0x25cdbd2bf399341f8fee22ecdb06682ac81fdc37"
// )

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

	// err = receivePacketOnEVM()
	// if err != nil {
	// 	log.Fatal(err)
	// }
}

// updateTendermintLightClient submits a MsgUpdateClient to the Tendermint light client on the EVM roll-up.
func updateTendermintLightClient() error {
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("addresses %v\n", addresses)

	ethClient, err := ethclient.Dial(ethereumRPC)
	if err != nil {
		return err
	}
	icsCore, err := icscore.NewContract(ethcommon.HexToAddress(addresses.ICSCore), ethClient)
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
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to prover: %w", err)
	}
	defer conn.Close()

	proverClient := proverclient.NewProverClient(conn)
	request := &proverclient.ProveStateTransitionRequest{ClientId: addresses.ICS07Tendermint}

	// Get state transition proof from Celestia prover with retry logic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var resp *proverclient.ProveStateTransitionResponse
	// var err error
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
	fmt.Printf("got resp %v\n", resp)

	// TODO: fetch vKey from some where
	vKey := [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}

	parsedABI, err := getFungibleTokenPacketDataABI()
	if err != nil {
		return err
	}

	msg := sp1ics07tendermint.IUpdateClientMsgsMsgUpdateClient{
		Sp1Proof: sp1ics07tendermint.ISP1MsgsSP1Proof{
			VKey:         vKey,
			PublicValues: resp.PublicValues,
			Proof:        resp.Proof,
		},
	}

	encoded, err := parsedABI.Pack(msg)
	if err != nil {
		return err
	}
	fmt.Printf("abi encoded updateMsg %v\n", encoded)
	fmt.Printf("Invoking icsCore.UpdateClient...\n")
	tx, err := icsCore.UpdateClient(getTransactOpts(faucet, eth), clientID, encoded)
	if err != nil {
		return err
	}
	fmt.Printf("icsCore.UpdateClient did not error\n")
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
	addresses, err := utils.ExtractDeployedContractAddresses()
	if err != nil {
		return err
	}

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
	ics26Contract, err := ics26router.NewContract(ethcommon.HexToAddress(addresses.ICS26Router), ethClient)
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

func getFungibleTokenPacketDataABI() (abi.Arguments, error) {
	parsedABI, err := ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	return parsedABI.Methods["abiPublicTypes"].Inputs, nil
}

// LibMetaData contains all meta data concerning the Lib contract.
//
//	var LibMetaData = &bind.MetaData{
//		ABI: "[{\"type\":\"function\",\"name\":\"DEFAULT_PORT_ID\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"FAILED_ACKNOWLEDGEMENT_JSON\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"IBC_DENOM_PREFIX\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ICS20_ENCODING\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ICS20_VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"SUCCESSFUL_ACKNOWLEDGEMENT_JSON\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"abiPublicTypes\",\"inputs\":[{\"name\":\"o1\",\"type\":\"tuple\",\"internalType\":\"structICS20Lib.FungibleTokenPacketData\",\"components\":[{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"sender\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"receiver\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"memo\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"newMsgSendPacketV1\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"msg_\",\"type\":\"tuple\",\"internalType\":\"structIICS20TransferMsgs.SendTransferMsg\",\"components\":[{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"receiver\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"sourceChannel\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"destPort\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"memo\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIICS26RouterMsgs.MsgSendPacket\",\"components\":[{\"name\":\"sourceChannel\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"payloads\",\"type\":\"tuple[]\",\"internalType\":\"structIICS26RouterMsgs.Payload[]\",\"components\":[{\"name\":\"sourcePort\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"destPort\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"version\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"encoding\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"value\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"toHexHash\",\"inputs\":[{\"name\":\"str\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"toIBCDenom\",\"inputs\":[{\"name\":\"fullDenomPath\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"error\",\"name\":\"ICS20InvalidAddress\",\"inputs\":[{\"name\":\"addr\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"ICS20InvalidAmount\",\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"StringsInsufficientHexLength\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]",
//	}
var ContractMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"updateClientProgramVkey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"membershipProgramVkey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"updateClientAndMembershipProgramVkey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"misbehaviourProgramVkey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_clientState\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_consensusState\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"ALLOWED_SP1_CLOCK_DRIFT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MEMBERSHIP_PROGRAM_VKEY\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MISBEHAVIOUR_PROGRAM_VKEY\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"UPDATE_CLIENT_AND_MEMBERSHIP_PROGRAM_VKEY\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"UPDATE_CLIENT_PROGRAM_VKEY\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"VERIFIER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISP1Verifier\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"abiPublicTypes\",\"inputs\":[{\"name\":\"o1\",\"type\":\"tuple\",\"internalType\":\"structIMembershipMsgs.MembershipOutput\",\"components\":[{\"name\":\"commitmentRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"kvPairs\",\"type\":\"tuple[]\",\"internalType\":\"structIMembershipMsgs.KVPair[]\",\"components\":[{\"name\":\"path\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"value\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]},{\"name\":\"o2\",\"type\":\"tuple\",\"internalType\":\"structIUpdateClientAndMembershipMsgs.UcAndMembershipOutput\",\"components\":[{\"name\":\"updateClientOutput\",\"type\":\"tuple\",\"internalType\":\"structIUpdateClientMsgs.UpdateClientOutput\",\"components\":[{\"name\":\"clientState\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ClientState\",\"components\":[{\"name\":\"chainId\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"trustLevel\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.TrustThreshold\",\"components\":[{\"name\":\"numerator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"denominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"}]},{\"name\":\"latestHeight\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"trustingPeriod\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"unbondingPeriod\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"isFrozen\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"zkAlgorithm\",\"type\":\"uint8\",\"internalType\":\"enumISP1Msgs.SupportedZkAlgorithm\"}]},{\"name\":\"trustedConsensusState\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ConsensusState\",\"components\":[{\"name\":\"timestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nextValidatorsHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"newConsensusState\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ConsensusState\",\"components\":[{\"name\":\"timestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nextValidatorsHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"time\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"trustedHeight\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"newHeight\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]}]},{\"name\":\"kvPairs\",\"type\":\"tuple[]\",\"internalType\":\"structIMembershipMsgs.KVPair[]\",\"components\":[{\"name\":\"path\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"value\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]},{\"name\":\"o3\",\"type\":\"tuple\",\"internalType\":\"structIUpdateClientMsgs.MsgUpdateClient\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"structISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]},{\"name\":\"o4\",\"type\":\"tuple\",\"internalType\":\"structIMembershipMsgs.MembershipProof\",\"components\":[{\"name\":\"proofType\",\"type\":\"uint8\",\"internalType\":\"enumIMembershipMsgs.MembershipProofType\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"o5\",\"type\":\"tuple\",\"internalType\":\"structIMembershipMsgs.SP1MembershipProof\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"structISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"trustedConsensusState\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ConsensusState\",\"components\":[{\"name\":\"timestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nextValidatorsHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]}]},{\"name\":\"o6\",\"type\":\"tuple\",\"internalType\":\"structIMembershipMsgs.SP1MembershipAndUpdateClientProof\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"structISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]},{\"name\":\"o7\",\"type\":\"tuple\",\"internalType\":\"structIMisbehaviourMsgs.MisbehaviourOutput\",\"components\":[{\"name\":\"clientState\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ClientState\",\"components\":[{\"name\":\"chainId\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"trustLevel\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.TrustThreshold\",\"components\":[{\"name\":\"numerator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"denominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"}]},{\"name\":\"latestHeight\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"trustingPeriod\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"unbondingPeriod\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"isFrozen\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"zkAlgorithm\",\"type\":\"uint8\",\"internalType\":\"enumISP1Msgs.SupportedZkAlgorithm\"}]},{\"name\":\"time\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"trustedHeight1\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"trustedHeight2\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"trustedConsensusState1\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ConsensusState\",\"components\":[{\"name\":\"timestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nextValidatorsHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"trustedConsensusState2\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ConsensusState\",\"components\":[{\"name\":\"timestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nextValidatorsHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]}]},{\"name\":\"o8\",\"type\":\"tuple\",\"internalType\":\"structIMisbehaviourMsgs.MsgSubmitMisbehaviour\",\"components\":[{\"name\":\"sp1Proof\",\"type\":\"tuple\",\"internalType\":\"structISP1Msgs.SP1Proof\",\"components\":[{\"name\":\"vKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"publicValues\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]}],\"outputs\":[],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getClientState\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.ClientState\",\"components\":[{\"name\":\"chainId\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"trustLevel\",\"type\":\"tuple\",\"internalType\":\"structIICS07TendermintMsgs.TrustThreshold\",\"components\":[{\"name\":\"numerator\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"denominator\",\"type\":\"uint8\",\"internalType\":\"uint8\"}]},{\"name\":\"latestHeight\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"trustingPeriod\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"unbondingPeriod\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"isFrozen\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"zkAlgorithm\",\"type\":\"uint8\",\"internalType\":\"enumISP1Msgs.SupportedZkAlgorithm\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getConsensusStateHash\",\"inputs\":[{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"membership\",\"inputs\":[{\"name\":\"msgMembership\",\"type\":\"tuple\",\"internalType\":\"structILightClientMsgs.MsgMembership\",\"components\":[{\"name\":\"proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proofHeight\",\"type\":\"tuple\",\"internalType\":\"structIICS02ClientMsgs.Height\",\"components\":[{\"name\":\"revisionNumber\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"revisionHeight\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"name\":\"path\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"value\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[{\"name\":\"timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"misbehaviour\",\"inputs\":[{\"name\":\"misbehaviourMsg\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"multicall\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[{\"name\":\"results\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateClient\",\"inputs\":[{\"name\":\"updateMsg\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumILightClientMsgs.UpdateResult\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeClient\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"view\"},{\"type\":\"error\",\"name\":\"AddressEmptyCode\",\"inputs\":[{\"name\":\"target\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"CannotHandleMisbehavior\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChainIdMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"actual\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"ClientStateMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"actual\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"type\":\"error\",\"name\":\"ConsensusStateHashMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"actual\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ConsensusStateNotFound\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ConsensusStateRootMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"actual\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"FailedCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeatureNotSupported\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FrozenClientState\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidMembershipProof\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"KeyValuePairNotInCache\",\"inputs\":[{\"name\":\"path\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"value\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"type\":\"error\",\"name\":\"LengthIsOutOfRange\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"min\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"max\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"MembershipProofKeyNotFound\",\"inputs\":[{\"name\":\"path\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}]},{\"type\":\"error\",\"name\":\"MembershipProofValueMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"actual\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"type\":\"error\",\"name\":\"ProofHeightMismatch\",\"inputs\":[{\"name\":\"expectedRevisionNumber\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"expectedRevisionHeight\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"actualRevisionNumber\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"actualRevisionHeight\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"type\":\"error\",\"name\":\"ProofIsInTheFuture\",\"inputs\":[{\"name\":\"now\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proofTimestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ProofIsTooOld\",\"inputs\":[{\"name\":\"now\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proofTimestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"TrustThresholdMismatch\",\"inputs\":[{\"name\":\"expectedNumerator\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"expectedDenominator\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"actualNumerator\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"actualDenominator\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"TrustingPeriodMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"actual\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"TrustingPeriodTooLong\",\"inputs\":[{\"name\":\"trustingPeriod\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"unbondingPeriod\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"UnbondingPeriodMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"actual\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"UnknownMembershipProofType\",\"inputs\":[{\"name\":\"proofType\",\"type\":\"uint8\",\"internalType\":\"uint8\"}]},{\"type\":\"error\",\"name\":\"UnknownZkAlgorithm\",\"inputs\":[{\"name\":\"algorithm\",\"type\":\"uint8\",\"internalType\":\"uint8\"}]},{\"type\":\"error\",\"name\":\"VerificationKeyMismatch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"actual\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]}]",
}
