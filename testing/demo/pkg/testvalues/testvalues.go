package testvalues

import (
	"time"

	"cosmossdk.io/math"

	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

const (
	// InitialBalance is the amount of tokens to give to each user at the start of the test.
	InitialBalance int64 = 1_000_000_000_000

	// TransferAmount is the default transfer amount
	TransferAmount int64 = 1_000_000_000

	// EnvKeyTendermintRPC Tendermint RPC URL.
	EnvKeyTendermintRPC = "TENDERMINT_RPC_URL"
	// EnvKeyEthRPC Ethereum RPC URL.
	EnvKeyEthRPC = "RPC_URL"
	// EnvKeyOperatorPrivateKey Private key used to submit transactions by the operator.
	EnvKeyOperatorPrivateKey = "PRIVATE_KEY"
	// EnvKeySp1Prover The prover type (local|network|mock).
	EnvKeySp1Prover = "SP1_PROVER"
	// EnvKeySp1PrivateKey Private key for the prover network.
	EnvKeySp1PrivateKey = "SP1_PRIVATE_KEY"
	// EnvKeyGenerateSolidityFixtures Generate fixtures for the solidity tests if set to true.
	EnvKeyGenerateSolidityFixtures = "GENERATE_SOLIDITY_FIXTURES"
	// EnvKeyGenerateSolidityFixtures Generate fixtures for the solidity tests if set to true.
	EnvKeyGenerateRustFixtures = "GENERATE_RUST_FIXTURES"
	// The log level for the Rust logger.
	EnvKeyRustLog = "RUST_LOG"
	// Address of the SP1ICS07Tendermint contract for operator.
	EnvKeyContractAddress = "CONTRACT_ADDRESS"

	// Log level for the Rust logger.
	EnvValueRustLog_Info = "info"
	// EnvValueSp1Prover_Network is the prover type for the network prover.
	EnvValueSp1Prover_Network = "network"
	// EnvValueSp1Prover_Mock is the prover type for the mock prover.
	EnvValueSp1Prover_Mock = "mock"
	// EnvValueGenerateFixtures_True is the value to set to generate fixtures for the solidity tests.
	EnvValueGenerateFixtures_True = "true"

	// EthTestnetTypePoW is the Ethereum testnet type for using a proof of work chain (anvil).
	EthTestnetTypePoW = "pow"
	// EthTestnetTypePoS is the Ethereum testnet type for using a proof of stake chain
	EthTestnetTypePoS = "pos"
	// EthTestnetTypeNone is the Ethereum testnet type for using no chain.
	EthTestnetTypeNone = "none"
	// EnvKeyEthTestnetType The Ethereum testnet type (pow|pos).
	EnvKeyEthTestnetType = "ETH_TESTNET_TYPE"
	// EnvE2EFacuetAddress The address of the faucet
	EnvKeyE2EFacuetAddress = "E2E_FAUCET_ADDRESS"

	// Sp1GenesisFilePath is the path to the genesis file for the SP1 chain.
	// This file is generated and then deleted by the test.
	Sp1GenesisFilePath = "scripts/genesis.json"
	// SolidityFixturesDir is the directory where the Solidity fixtures are stored.
	SolidityFixturesDir = "test/solidity-ibc/fixtures/"
	// SP1ICS07FixturesDir is the directory where the SP1ICS07 fixtures are stored.
	SP1ICS07FixturesDir = "test/sp1-ics07/fixtures"
	// RustFixturesDir is the directory where the Rust fixtures are stored.
	RustFixturesDir = "packages/ethereum-light-client/src/test_utils/fixtures"
	// RelayerConfigFilePath is the path to generate the relayer config file.
	RelayerConfigFilePath = "programs/relayer/config.json"
	// E2EDeployScriptPath is the path to the E2E deploy script.
	E2EDeployScriptPath = "scripts/E2ETestDeploy.s.sol:E2ETestDeploy"
	// SP1ICS07DeployScriptPath is the path to the SP1ICS07 deploy script.
	SP1ICS07DeployScriptPath = "scripts/SP1ICS07Tendermint.s.sol:SP1TendermintScript"

	// DefaultGovV1ProposalTokenAmount is the default amount of tokens used to submit a proposal.
	DefaultGovV1ProposalTokenAmount = 500_000_000

	// IbcCommitmentSlotHex is the storage slot in the IBC solidity contract for the IBC commitments.
	IbcCommitmentSlotHex = "0x1"
)

var (
	// MaxDepositPeriod Maximum period to deposit on a proposal.
	// This value overrides the default value in the gov module using the `modifyGovV1AppState` function.
	MaxDepositPeriod = time.Second * 10
	// VotingPeriod Duration of the voting period.
	// This value overrides the default value in the gov module using the `modifyGovV1AppState` function.
	VotingPeriod = time.Second * 30

	// StartingEthBalance is the amount of ETH to give to each user at the start of the test.
	StartingEthBalance = math.NewInt(2 * 1000000000000000000)

	// DefaultTrustLevel is the trust level used by the SP1ICS07Tendermint contract.
	DefaultTrustLevel = ibctm.Fraction{Numerator: 2, Denominator: 3}.ToTendermint()

	// DefaultTrustPeriod is the trust period used by the SP1ICS07Tendermint contract.
	DefaultTrustPeriod = 1209669
)
