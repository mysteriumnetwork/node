/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/registry"
	"github.com/mysteriumnetwork/payments/registry/generated"
)

// IdentityRegistry checks whenever given identity is registered
type IdentityRegistry interface {
	IsRegistered(identity common.Address) (bool, error)
}

// RegistrationDataProvider provides registration information for given identity required to register it on blockchain
type RegistrationDataProvider interface {
	ProvideRegistrationData(identity common.Address) (*registry.RegistrationData, error)
}

type keystoreRegistrationDataProvider struct {
	ks *keystore.KeyStore
}

func (kpg *keystoreRegistrationDataProvider) ProvideRegistrationData(identity common.Address) (*registry.RegistrationData, error) {
	identityHolder := registry.FromKeystore(kpg.ks, identity)

	return registry.CreateRegistrationData(identityHolder)
}

// NewRegistrationDataProvider creates registration data provider backed up by identity which is managed by keystore
func NewRegistrationDataProvider(ks *keystore.KeyStore) RegistrationDataProvider {
	return &keystoreRegistrationDataProvider{
		ks: ks,
	}
}

// NewIdentityRegistry creates identity registry service which uses blockchain for information
func NewIdentityRegistry(contractCaller bind.ContractCaller, registryAddress common.Address) (IdentityRegistry, error) {
	contract, err := generated.NewIdentityRegistryCaller(registryAddress, contractCaller)
	if err != nil {
		return nil, err
	}

	return &generated.IdentityRegistryCallerSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}, nil
}
