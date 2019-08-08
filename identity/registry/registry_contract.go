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
	"context"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/bindings"

	"github.com/mysteriumnetwork/node/identity"
)

const logPrefix = "[registry] "

// NewIdentityRegistryContract creates identity registry service which uses blockchain for information
func NewIdentityRegistryContract(contractBackend bind.ContractBackend, registryAddress, accountantAddress common.Address) (*contractRegistry, error) {
	contract, err := bindings.NewRegistryCaller(registryAddress, contractBackend)
	if err != nil {
		return nil, err
	}

	contractSession := &bindings.RegistryCallerSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}

	filterer, err := bindings.NewRegistryFilterer(registryAddress, contractBackend)
	if err != nil {
		return nil, err
	}
	return &contractRegistry{
		contractSession,
		filterer,
		accountantAddress,
	}, nil
}

type contractRegistry struct {
	contractSession   *bindings.RegistryCallerSession
	filterer          *bindings.RegistryFilterer
	accountantAddress common.Address
}

func (registry *contractRegistry) IsRegistered(id identity.Identity) (bool, error) {
	return registry.contractSession.IsRegistered(
		common.HexToAddress(id.Address),
	)
}

// RegistrationEvent describes registration events
type RegistrationEvent int

// Possible registration events
const (
	Registered RegistrationEvent = 0
	Cancelled  RegistrationEvent = 1
)

// SubscribeToRegistrationEvent returns registration event if given providerAddress was registered within payments contract
func (registry *contractRegistry) SubscribeToRegistrationEvent(id identity.Identity) (
	registrationEvent chan RegistrationEvent,
	unsubscribe func(),
) {
	registrationEvent = make(chan RegistrationEvent)

	stopLoop := make(chan bool)
	unsubscribe = func() {
		// cancel (stop) identity registration loop
		stopLoop <- true
	}

	identities := []common.Address{
		registry.accountantAddress,
	}

	filterOps := &bind.FilterOpts{
		Start:   0,
		End:     nil,
		Context: context.Background(),
	}

	go func() {
		for {
			select {
			case <-stopLoop:
				registrationEvent <- Cancelled
				return
			// TODO: adjust  this time to something more appropriate
			case <-time.After(1 * time.Second):
				logIterator, err := registry.filterer.FilterRegisteredIdentity(filterOps, identities)
				if err != nil {
					registrationEvent <- Cancelled
					log.Error(logPrefix, err)
					return
				}
				if logIterator == nil {
					registrationEvent <- Cancelled
					return
				}
				for {
					next := logIterator.Next()
					if next {
						ev := *logIterator.Event
						if strings.ToLower(ev.IdentityHash.Hex()) == strings.ToLower(id.Address) {
							registrationEvent <- Registered
							return
						}
					} else {
						err = logIterator.Error()
						if err != nil {
							log.Error(logPrefix, err)
						}
						break
					}
				}
			}
		}
	}()
	return
}
