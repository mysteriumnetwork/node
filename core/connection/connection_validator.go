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

package connection

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

type consumerBalanceGetter interface {
	GetBalance(ID identity.Identity) uint64
}

type unlockChecker interface {
	IsUnlocked(ID string) bool
}

// Validator validates pre connection conditions.
type Validator struct {
	consumerBalanceGetter consumerBalanceGetter
	unlockChecker         unlockChecker
}

// NewValidator returns a new instance of connection validator.
func NewValidator(consumerBalanceGetter consumerBalanceGetter, unlockChecker unlockChecker) *Validator {
	return &Validator{
		consumerBalanceGetter: consumerBalanceGetter,
		unlockChecker:         unlockChecker,
	}
}

// validateBalance checks if consumer has enough money for given proposal.
func (v *Validator) validateBalance(consumerID identity.Identity, proposal market.ServiceProposal) bool {
	if proposal.PaymentMethodType == "" || proposal.PaymentMethod == nil {
		return true
	}

	proposalPrice := proposal.PaymentMethod.GetPrice()
	balance := v.consumerBalanceGetter.GetBalance(consumerID)
	return balance >= proposalPrice.Amount
}

// isUnlocked checks if the identity is unlocked or not.
func (v *Validator) isUnlocked(consumerID identity.Identity) bool {
	return v.unlockChecker.IsUnlocked(consumerID.Address)
}

// Validate checks whether the pre-connection conditions are fulfilled.
func (v *Validator) Validate(consumerID identity.Identity, proposal market.ServiceProposal) error {
	if !v.isUnlocked(consumerID) {
		return ErrUnlockRequired
	}

	if !v.validateBalance(consumerID, proposal) {
		return ErrInsufficientBalance
	}

	return nil
}
