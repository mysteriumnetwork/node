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

// Handler describes a beneficiary handler.
type Handler interface {
	SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error
	GetBeneficiary(identity common.Address) (common.Address, error)
}

type storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

type multiChainBC interface {
	GetBeneficiary(chainID int64, registryAddress, identity common.Address) (common.Address, error)
}

type settler interface {
	SettleWithBeneficiary(chainID int64, id identity.Identity, beneficiary, hermesID common.Address) error
}

type addressProvider interface {
	GetActiveHermes(chainID int64) (common.Address, error)
	GetRegistryAddress(chainID int64) (common.Address, error)
	GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error)
}

// NewHandler returns a new beneficiary handler according to the given chain.
func NewHandler(currentChain int64, ad addressProvider, st storage, bc multiChainBC, set settler) Handler {
	if currentChain == metadata.DefaultNetwork.Chain1.ChainID {
		return newL1Handler(currentChain, ad, bc, set)
	}

	return newL2Handler(currentChain, ad, st, set)
}
