//nolint:govet
package groth16

import (
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var _ exported.ClientMessage = (*Header)(nil)

// ClientType defines that the Header is a Tendermint consensus algorithm
func (h Header) ClientType() string {
	return Groth16ClientType
}

// GetHeight returns the current height. It returns 0 if the tendermint
// header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetHeight() exported.Height {
	return clienttypes.NewHeight(0, uint64(h.NewHeight))
}

func (h Header) ValidateBasic() error {
	return nil
}
