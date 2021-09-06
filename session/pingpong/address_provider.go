/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
}

// NewAddressProvider returns a new instance of AddressProvider.
func NewAddressProvider(multichainAddressKeeper *client.MultiChainAddressKeeper) *AddressProvider {
	return &AddressProvider{
		MultiChainAddressKeeper: multichainAddressKeeper,
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
