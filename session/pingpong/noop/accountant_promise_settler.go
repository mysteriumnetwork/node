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

// NoopAccountantPromiseSettler doesn't do much.
type NoopAccountantPromiseSettler struct {
}

// Subscribe does nothing.
func (n *NoopAccountantPromiseSettler) Subscribe() error {
	return nil
}

// GetEarnings returns an empty state.
func (n *NoopAccountantPromiseSettler) GetEarnings(_ identity.Identity) event.Earnings {
	return event.Earnings{}
}

// ForceSettle does nothing.
func (n *NoopAccountantPromiseSettler) ForceSettle(_ identity.Identity, _ common.Address) error {
	return nil
}
