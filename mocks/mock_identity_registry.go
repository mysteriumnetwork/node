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

package mocks

import (
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
)

// IdentityRegistry is a fake identity registry.
type IdentityRegistry struct {
	Status registry.RegistrationStatus
}

// Subscribe does nothing.
func (i IdentityRegistry) Subscribe(_ eventbus.Subscriber) error {
	return nil
}

// GetRegistrationStatus returns a pre-defined RegistrationStatus.
func (i IdentityRegistry) GetRegistrationStatus(_ int64, _ identity.Identity) (registry.RegistrationStatus, error) {
	return i.Status, nil
}
