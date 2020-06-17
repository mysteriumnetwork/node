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
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
)

// NoopHermesPromiseSettler doesn't do much.
type NoopHermesPromiseSettler struct {
}

// Subscribe does nothing.
func (n *NoopHermesPromiseSettler) Subscribe() error {
	return nil
}

// GetEarnings returns an empty state.
func (n *NoopHermesPromiseSettler) GetEarnings(_ identity.Identity) event.Earnings {
	return event.Earnings{}
}

// ForceSettle does nothing.
func (n *NoopHermesPromiseSettler) ForceSettle(_ identity.Identity, _ common.Address) error {
	return nil
}

// SettleWithBeneficiary does nothing.
func (n *NoopHermesPromiseSettler) SettleWithBeneficiary(_ identity.Identity, _, _ common.Address) error {
	return nil
}

// GetHermesFee does absolutely nothing.
func (n *NoopHermesPromiseSettler) GetHermesFee() (uint16, error) {
	return 0, nil
}
