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
	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	payments_identity "github.com/mysteriumnetwork/payments/identity"
	"github.com/mysteriumnetwork/payments/registry"
)

// FakeRegister fake register
type FakeRegister struct {
	RegistrationEventExists bool
	Registered              bool
}

// IsRegistered returns fake identity registration status within payments contract
func (register *FakeRegister) IsRegistered(id identity.Identity) (bool, error) {
	return register.Registered, nil
}

// SubscribeToRegistrationEvent returns fake registration event if given providerAddress was registered within payments contract
func (register *FakeRegister) SubscribeToRegistrationEvent(providerAddress common.Address) (registrationEvent chan RegistrationEvent, unsubscribe func()) {
	log.Info("fake SubscribeToRegistrationEvent called ")
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

// FakeRegistrationDataProvider fake registration data provider
type FakeRegistrationDataProvider struct {
}

// Fake public key parts for tests
const (
	testPubPart1 = "0xFA001122334455667788990011223344556677889900112233445566778899AF"
	testPubPart2 = "0xDE001122334455667788990011223344556677889900112233445566778899AD"
)

// ProvideRegistrationData provides fake registration data
func (m *FakeRegistrationDataProvider) ProvideRegistrationData(id identity.Identity) (*registry.RegistrationData, error) {
	registrationData := &registry.RegistrationData{
		PublicKey: registry.PublicKeyParts{
			Part1: common.FromHex(testPubPart1),
			Part2: common.FromHex(testPubPart2),
		},
		Signature: &payments_identity.DecomposedSignature{
			R: [32]byte{1},
			S: [32]byte{2},
			V: 27,
		},
	}
	return registrationData, nil
}
