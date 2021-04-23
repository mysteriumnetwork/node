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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
)

// Provider describes a beneficiary provider.
type Provider interface {
	GetBeneficiary(identity common.Address) (common.Address, error)
}

// NewProvider returns a new beneficiary handler according to the given chain.
func NewProvider(currentChain int64, ad addressProvider, st storage, bc multiChainBC) Provider {
	if currentChain == metadata.DefaultNetwork.Chain1.ChainID {
		return newL1Provider(currentChain, ad, bc)
	}

	return newL2Provider(currentChain, ad, st)
}

// l1Provider handles getting beneficiary in L1 chains.
type l1Provider struct {
	bc      multiChainBC
	ad      addressProvider
	chainID int64
}

func newL1Provider(chainID int64, ad addressProvider, bc multiChainBC) *l1Provider {
	return &l1Provider{
		chainID: chainID,
		bc:      bc,
		ad:      ad,
	}
}

// GetBeneficiary looks up beneficiary address in the blockchain.
func (b *l1Provider) GetBeneficiary(identity common.Address) (common.Address, error) {
	registryAddr, err := b.ad.GetRegistryAddress(b.chainID)
	if err != nil {
		return common.Address{}, err
	}

	return b.bc.GetBeneficiary(b.chainID, registryAddr, identity)
}

// l2Provider handles getting beneficiary in L2 chains.
type l2Provider struct {
	set settler
	st  storage

	chainID int64
	ad      addressProvider
}

func newL2Provider(chainID int64, ad addressProvider, st storage) *l2Provider {
	return &l2Provider{
		st: st,

		chainID: chainID,
		ad:      ad,
	}
}

// GetBeneficiary returns an already saved beneficiary.
func (b *l2Provider) GetBeneficiary(id common.Address) (common.Address, error) {
	var addr string
	err := b.st.GetValue(storageBucket, storageKey(b.chainID, id.Hex()), &addr)
	if err != nil {
		// TODO: move this check to hermes channel repository
		if err.Error() == "not found" {
			// return generated consumer channel address then as that is the default.
			return b.ad.GetChannelAddress(b.chainID, identity.FromAddress(id.Hex()))
		}
	}
	return common.HexToAddress(addr), err
}
