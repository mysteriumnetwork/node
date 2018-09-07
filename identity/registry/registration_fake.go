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
	"github.com/mysteriumnetwork/node/identity"
	payments_identity "github.com/mysteriumnetwork/payments/identity"
	"github.com/mysteriumnetwork/payments/registry"
)

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
