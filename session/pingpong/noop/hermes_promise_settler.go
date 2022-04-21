/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package noop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
)

// NoopHermesPromiseSettler doesn't do much.
type NoopHermesPromiseSettler struct {
}

// ForceSettle does nothing.
func (n *NoopHermesPromiseSettler) ForceSettle(chainID int64, _ identity.Identity, _ ...common.Address) error {
	return nil
}

// SettleIntoStake does nothing.
func (n *NoopHermesPromiseSettler) SettleIntoStake(chainID int64, providerID identity.Identity, accountantID ...common.Address) error {
	return nil
}

// SettleWithBeneficiary does nothing.
func (n *NoopHermesPromiseSettler) SettleWithBeneficiary(chainID int64, _ identity.Identity, _, _ common.Address) error {
	return nil
}

// GetHermesFee does absolutely nothing.
func (n *NoopHermesPromiseSettler) GetHermesFee(chainID int64, _ common.Address) (uint16, error) {
	return 0, nil
}

// Withdraw does absolutely nothing.
func (n *NoopHermesPromiseSettler) Withdraw(fromChainID int64, toChainID int64, providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int) error {
	return nil
}

// CheckLatestWithdrawal does absolutely nothing.
func (n *NoopHermesPromiseSettler) CheckLatestWithdrawal(chainID int64, providerID identity.Identity, hermesID common.Address) (*big.Int, string, error) {
	return nil, "", nil
}

// RetryWithdrawLatest does absolutely nothing.
func (n *NoopHermesPromiseSettler) RetryWithdrawLatest(chainID int64, amountToWithdraw *big.Int, chid string, beneficiary common.Address, providerID identity.Identity) error {
	return nil
}
