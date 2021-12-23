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

package beneficiary

import (
	"github.com/ethereum/go-ethereum/common"
)

// Provider describes a beneficiary provider.
type Provider struct {
	bc      multiChainBC
	ad      addressProvider
	chainID int64
}

// NewProvider returns a new beneficiary handler according to the given chain.
func NewProvider(currentChain int64, ad addressProvider, st storage, bc multiChainBC) *Provider {
	return &Provider{
		chainID: currentChain,
		bc:      bc,
		ad:      ad,
	}
}

// GetBeneficiary looks up beneficiary address in the blockchain.
func (b *Provider) GetBeneficiary(identity common.Address) (common.Address, error) {
	registryAddr, err := b.ad.GetRegistryAddress(b.chainID)
	if err != nil {
		return common.Address{}, err
	}

	return b.bc.GetBeneficiary(b.chainID, registryAddr, identity)
}
