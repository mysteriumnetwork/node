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
)

// Saver saves a given beneficiary and tracks its
// progress.
type Saver struct {
	set     settler
	ad      addressProvider
	chainID int64
	*beneficiaryChangeKeeper
}

type storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

type multiChainBC interface {
	GetBeneficiary(chainID int64, registryAddress, identity common.Address) (common.Address, error)
}

type settler interface {
	SettleWithBeneficiary(chainID int64, id identity.Identity, beneficiary common.Address, hermeses []common.Address) error
}

type addressProvider interface {
	GetActiveHermes(chainID int64) (common.Address, error)
	GetRegistryAddress(chainID int64) (common.Address, error)
	GetActiveChannelAddress(chainID int64, id common.Address) (common.Address, error)
}

// NewSaver returns a new beneficiary saver according to the given chain.
func NewSaver(currentChain int64, ad addressProvider, st storage, bc multiChainBC, set settler) *Saver {
	return &Saver{
		chainID:                 currentChain,
		set:                     set,
		ad:                      ad,
		beneficiaryChangeKeeper: newBeneficiaryChangeKeeper(currentChain, st),
	}
}

// SettleAndSaveBeneficiary executes a settlement transaction saving the beneficiary to the blockchain.
func (b *Saver) SettleAndSaveBeneficiary(id identity.Identity, hermeses []common.Address, beneficiary common.Address) error {
	return b.executeWithStatusTracking(id, beneficiary, func() error {
		return b.set.SettleWithBeneficiary(b.chainID, id, beneficiary, hermeses)
	})
}
