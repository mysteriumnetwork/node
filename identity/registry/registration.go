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
	"context"
	"errors"
	"time"

	"github.com/MysteriumNetwork/payments/registry"
	"github.com/MysteriumNetwork/payments/registry/generated"
	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

var ErrNoIdentityRegisteredTimeout = errors.New("no identity registration for long time, exiting")

type Register struct {
	callerSession *generated.IdentityRegistryCallerSession
	filterer      *generated.IdentityRegistryFilterer
}

// IdentityRegistry enables identity registration actions
type IdentityRegistry interface {
	IsRegistered(identity common.Address) (bool, error)
	WaitForRegistrationEvent(providerAddress common.Address, registeredEvent chan bool)
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
func NewIdentityRegistry(contractBackend bind.ContractBackend, registryAddress common.Address) (IdentityRegistry, error) {
	contract, err := generated.NewIdentityRegistryCaller(registryAddress, contractBackend)
	if err != nil {
		return nil, err
	}

	filterer, err := generated.NewIdentityRegistryFilterer(registryAddress, contractBackend)
	if err != nil {
		return nil, err
	}

	return &Register{
		&generated.IdentityRegistryCallerSession{
			Contract: contract,
			CallOpts: bind.CallOpts{
				Pending: false, //we want to find out true registration status - not pending transactions
			},
		},
		filterer,
	}, nil
}

func (register *Register) IsRegistered(identity common.Address) (bool, error) {
	return register.callerSession.IsRegistered(identity)
}

func (register *Register) WaitForRegistrationEvent(providerAddress common.Address, registeredEvent chan bool) {
	identities := []common.Address{providerAddress}

	filterOps := &bind.FilterOpts{
		Start:   0,
		End:     nil,
		Context: context.Background(),
	}

	for {
		logIterator, err := register.filterer.FilterRegistered(filterOps, identities)
		if err != nil {
			log.Error(err)
		}

		for {
			next := logIterator.Next()
			if !next {
				err = logIterator.Error()
				if err != nil {
					log.Error(err)
				}
				break
			}
			log.Info("got identity registration event")
			registeredEvent <- true
		}
		time.Sleep(100 * time.Millisecond)
		log.Info("no identity registration, sleeping for 100ms: ")
	}
}
