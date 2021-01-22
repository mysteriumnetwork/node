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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
)

// l2Handler handles beneficiary in L2 chains.
// Beneficiary is stored and retrieved from the given storage.
type l2Handler struct {
	set settler
	st  storage

	chainID int64
	ad      addressProvider
}

const storageBucket = "beneficiary-bucket"

func newL2Handler(chainID int64, ad addressProvider, st storage, set settler) *l2Handler {
	return &l2Handler{
		set: set,
		st:  st,

		chainID: chainID,
		ad:      ad,
	}
}

// SettleAndSaveBeneficiary settles beneficiary set to users own payments channel address.
// The given beneficiary is saved to the database and later retrieved from there.
func (b *l2Handler) SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error {
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

	return b.st.SetValue(storageBucket, b.storageKey(id.Address), beneficiary.Hex())
}

// GetBeneficiary returns an already saved beneficiary.
func (b *l2Handler) GetBeneficiary(identity common.Address) (common.Address, error) {
	var addr string
	err := b.st.GetValue(storageBucket, b.storageKey(identity.Hex()), &addr)

	return common.HexToAddress(addr), err
}

func (b *l2Handler) storageKey(identityHex string) string {
	return fmt.Sprintf("%d|%s", b.chainID, identityHex)
}
