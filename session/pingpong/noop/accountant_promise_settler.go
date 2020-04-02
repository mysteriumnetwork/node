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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
)

// NoopAccountantPromiseSettler doesn't do much.
type NoopAccountantPromiseSettler struct {
}

// Subscribe does nothing.
func (n *NoopAccountantPromiseSettler) Subscribe() error {
	return nil
}

// SettlementState returns an empty state.
func (n *NoopAccountantPromiseSettler) SettlementState(id identity.Identity) pingpong.SettlementState {
	return pingpong.SettlementState{
		Channel:     client.ProviderChannel{},
		LastPromise: crypto.Promise{},
	}
}

// ForceSettle does nothing.
func (n *NoopAccountantPromiseSettler) ForceSettle(_, _ identity.Identity) error {
	return nil
}
