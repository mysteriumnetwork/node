/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package registry

import (
	"github.com/ethereum/go-ethereum/common"
)

// FakeRegister fake register
type FakeRegister struct {
	RegistrationEventExists bool
}

// IsRegistered returns fake identity registration status within payments contract
func (register *FakeRegister) IsRegistered(identity common.Address) (bool, error) {
	return true, nil
}

// SubscribeToRegistrationEvent returns fake registration event if given providerAddress was registered within payments contract
func (register *FakeRegister) SubscribeToRegistrationEvent(providerAddress common.Address) (registrationEvent chan RegistrationEvent, unsubscribe func()) {
	registrationEvent = make(chan RegistrationEvent)
	unsubscribe = func() {
		registrationEvent <- Cancelled
	}
	go func() {
		if register.RegistrationEventExists {
			registrationEvent <- Registered
		}
	}()
	return registrationEvent, unsubscribe
}
