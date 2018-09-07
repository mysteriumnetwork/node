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
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/registry"
)

// NewRegistrationDataProvider creates registration data provider backed up by identity which is managed by keystore
func NewRegistrationDataProvider(ks *keystore.KeyStore) *keystoreRegistrationDataProvider {
	return &keystoreRegistrationDataProvider{
		ks: ks,
	}
}

type keystoreRegistrationDataProvider struct {
	ks *keystore.KeyStore
}

func (kpg *keystoreRegistrationDataProvider) ProvideRegistrationData(id identity.Identity) (*registry.RegistrationData, error) {
	identityHolder := registry.FromKeystore(kpg.ks, common.HexToAddress(id.Address))

	return registry.CreateRegistrationData(identityHolder)
}
