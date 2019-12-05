/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysteriumnetwork/payments/crypto"
)

// ChannelAddressCalculator calculates the
type ChannelAddressCalculator struct {
	accountantSCAddress   common.Address
	channelImplementation common.Address
	registryAddress       common.Address
}

// NewChannelAddressCalculator returns a new instaance of channel address calculator
func NewChannelAddressCalculator(accountantSCAddress, channelImplementation, registryAddress common.Address) *ChannelAddressCalculator {
	return &ChannelAddressCalculator{
		accountantSCAddress:   accountantSCAddress,
		channelImplementation: channelImplementation,
		registryAddress:       registryAddress,
	}
}

// GetChannelAddress returns channel id
func (cac *ChannelAddressCalculator) GetChannelAddress(ID identity.Identity) (common.Address, error) {
	addr, err := crypto.GenerateChannelAddress(ID.Address, cac.accountantSCAddress.Hex(), cac.registryAddress.Hex(), cac.channelImplementation.Hex())
	return common.HexToAddress(addr), err
}
