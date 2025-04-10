package groth16

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.ClientMessage = (*Header)(nil)

// ClientType returns the Groth16 client type.
func (h *Header) ClientType() string {
	return Groth16ClientType
}

// GetHeight returns the EVM height of this header.
func (h *Header) GetHeight() exported.Height {
	return clienttypes.NewHeight(0, uint64(h.NewHeight))
}

// ValidateBasic is a no-op.
func (h *Header) ValidateBasic() error {
	return nil
}
