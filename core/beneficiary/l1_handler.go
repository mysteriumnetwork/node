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

// l1Handler handles saving beneficiary in L1 chains.
// Beneficiary is stored and retrieved from the blockchain.
type l1Handler struct {
	bc      multiChainBC
	set     settler
	ad      addressProvider
	chainID int64
}

func newL1Handler(chainID int64, ad addressProvider, bc multiChainBC, set settler) *l1Handler {
	return &l1Handler{
		chainID: chainID,
		bc:      bc,
		set:     set,
		ad:      ad,
	}
}

// SettleAndSaveBeneficiary executes a settlement transaction saving the beneficiary to the blockchain.
func (b *l1Handler) SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error {
	return b.set.SettleWithBeneficiary(b.chainID, id, beneficiary, b.hermesID)
}

// GetBeneficiary looks up beneficiary address in the blockchain.
func (b *l1Handler) GetBeneficiary(identity common.Address) (common.Address, error) {
	return b.bc.GetBeneficiary(b.chainID, b.registryAddress, identity)
}
