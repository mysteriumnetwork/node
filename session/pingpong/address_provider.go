package pingpong

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
)

// AddressProvider can calculate channel addresses as well as provide SC addresses for various chains.
type AddressProvider struct {
	*client.MultiChainAddressKeeper
	transactorAddress common.Address
}

// NewAddressProvider returns a new instance of AddressProvider.
func NewAddressProvider(multichainAddressKeeper *client.MultiChainAddressKeeper, transactorAddress common.Address) *AddressProvider {
	return &AddressProvider{
		MultiChainAddressKeeper: multichainAddressKeeper,
		transactorAddress:       transactorAddress,
	}
}

// GetChannelAddress calculates the channel address for the given chain.
func (ap *AddressProvider) GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error) {
	hermes, err := ap.MultiChainAddressKeeper.GetActiveHermes(chainID)
	if err != nil {
		return common.Address{}, nil
	}
	registry, err := ap.MultiChainAddressKeeper.GetRegistryAddress(chainID)
	if err != nil {
		return common.Address{}, nil
	}
	channel, err := ap.MultiChainAddressKeeper.GetChannelImplementation(chainID)
	if err != nil {
		return common.Address{}, nil
	}

	addr, err := crypto.GenerateChannelAddress(id.Address, hermes.Hex(), registry.Hex(), channel.Hex())
	return common.HexToAddress(addr), err
}

// GetArbitraryChannelAddress calculates a channel address from the given params.
func (ap *AddressProvider) GetArbitraryChannelAddress(hermes, registry, channel common.Address, id identity.Identity) (common.Address, error) {
	addr, err := crypto.GenerateChannelAddress(id.Address, hermes.Hex(), registry.Hex(), channel.Hex())
	return common.HexToAddress(addr), err
}

// GetTransactorAddress returns the transactor address.
func (ap *AddressProvider) GetTransactorAddress() common.Address {
	return ap.transactorAddress
}
