package groth16

import (
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModuleBasic = (*AppModuleBasic)(nil)
	_ appmodule.AppModule   = (*AppModule)(nil)
)

// AppModuleBasic defines the basic application module used by the groth16 light client.
// Only the RegisterInterfaces function needs to be implemented. All other function perform
// a no-op.
type AppModuleBasic struct{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModuleBasic) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModuleBasic) IsAppModule() {}

// Name returns the groth16 module name.
func (AppModuleBasic) Name() string {
	return ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterLegacyAminoCodec is a no-op. The groth16 client does not support amino.
func (AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any. This allows core IBC
// to unmarshal groth16 light client types.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	RegisterInterfaces(registry)
}

// DefaultGenesis is a no-op. Genesis is not supported for the groth16 light client.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ValidateGenesis is a no-op. Genesis is not supported for the groth16 light client.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	return nil
}

// RegisterGRPCGatewayRoutes is a no-op.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

// GetTxCmd is a no-op.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd is a no-op.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// AppModule is the application module for the groth16 client module
type AppModule struct {
	AppModuleBasic
	lightClientModule LightClientModule
}

// NewAppModule returns a new groth16 client module.
func NewAppModule(lightClientModule LightClientModule) AppModule {
	return AppModule{
		lightClientModule: lightClientModule,
	}
}
