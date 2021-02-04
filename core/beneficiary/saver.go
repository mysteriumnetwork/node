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

// Saver describes a beneficiary saver.
type Saver interface {
	SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error
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

// NewSaver returns a new beneficiary saver according to the given chain.
func NewSaver(currentChain int64, ad addressProvider, st storage, bc multiChainBC, set settler) Saver {
	if currentChain == metadata.DefaultNetwork.Chain1.ChainID {
		return newL1Saver(currentChain, ad, bc, set)
	}

	return newL2Saver(currentChain, ad, st, set)
}

// l1Saver handles saving beneficiary in L1 chains.
type l1Saver struct {
	set     settler
	ad      addressProvider
	chainID int64
}

func newL1Saver(chainID int64, ad addressProvider, bc multiChainBC, set settler) *l1Saver {
	return &l1Saver{
		chainID: chainID,
		set:     set,
		ad:      ad,
	}
}

// SettleAndSaveBeneficiary executes a settlement transaction saving the beneficiary to the blockchain.
func (b *l1Saver) SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error {
	hermesID, err := b.ad.GetActiveHermes(b.chainID)
	if err != nil {
		return err
	}

	return b.set.SettleWithBeneficiary(b.chainID, id, beneficiary, hermesID)
}

// l2Saver handles saving beneficiary in L2 chains.
type l2Saver struct {
	set settler
	st  storage

	chainID int64
	ad      addressProvider
}

func newL2Saver(chainID int64, ad addressProvider, st storage, set settler) *l2Saver {
	return &l2Saver{
		set: set,
		st:  st,

		chainID: chainID,
		ad:      ad,
	}
}

// SettleAndSaveBeneficiary settles beneficiary set to users own payments channel address.
// The given beneficiary is saved to the database and later retrieved from there.
func (b *l2Saver) SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error {
	hermesID, err := b.ad.GetActiveHermes(b.chainID)
	if err != nil {
		return err
	}

	addr, err := b.ad.GetChannelAddress(b.chainID, id)
	if err != nil {
		return err
	}

	if err := b.set.SettleWithBeneficiary(b.chainID, id, addr, hermesID); err != nil {
		return err
	}

	return b.st.SetValue(storageBucket, storageKey(b.chainID, id.Address), beneficiary.Hex())
}
