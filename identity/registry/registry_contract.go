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
	"encoding/hex"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type registryStorage interface {
	Store(status StoredRegistrationStatus) error
	Get(identity identity.Identity) (StoredRegistrationStatus, error)
	GetAll() ([]StoredRegistrationStatus, error)
	UpdateStatus(identity identity.Identity, status RegistrationStatus) error
}

// NewIdentityRegistryContract creates identity registry service which uses blockchain for information
func NewIdentityRegistryContract(contractBackend bind.ContractBackend, registryAddress, accountantAddress common.Address, registryStorage registryStorage, publisher eventbus.Publisher) (*contractRegistry, error) {
	log.Info().Msgf("using registryAddress %v accountantAddress %v", registryAddress.Hex(), accountantAddress.Hex())
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

	accountantContract, err := bindings.NewAccountantImplementationCaller(accountantAddress, contractBackend)
	if err != nil {
		return nil, err
	}
	accountantSession := &bindings.AccountantImplementationCallerSession{
		Contract: accountantContract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}

	filterer, err := bindings.NewRegistryFilterer(registryAddress, contractBackend)
	if err != nil {
		return nil, err
	}
	return &contractRegistry{
		contractSession:   contractSession,
		accountantSession: accountantSession,
		filterer:          filterer,
		accountantAddress: accountantAddress,
		storage:           registryStorage,
		stop:              make(chan struct{}),
		publisher:         publisher,
	}, nil
}

func (registry *contractRegistry) Subscribe(eb eventbus.Subscriber) error {
	err := eb.SubscribeAsync(event.Topic, registry.handleNodeEvent)
	if err != nil {
		return err
	}

	return eb.Subscribe(RegistrationTopic, registry.handleRegistrationEvent)
}

func (registry *contractRegistry) handleNodeEvent(ev event.Payload) {
	log.Debug().Msgf("event received %v", ev)
	if ev.Status == event.StatusStarted {
		registry.handleStart()
		return
	}
	if ev.Status == event.StatusStopped {
		registry.handleStop()
		return
	}
}

func (registry *contractRegistry) storeStatus(status StoredRegistrationStatus) error {
	return registry.storage.Store(status)
}

func (registry *contractRegistry) GetRegistrationStatus(id identity.Identity) (RegistrationStatus, error) {
	status, err := registry.getStatus(id)
	if err != nil {
		if err == ErrNotFound {
			// TODO: this might need to be checked on BC then
			return Unregistered, nil
		}
		return Unregistered, errors.Wrap(err, "could not check status in local db")
	}

	return status.RegistrationStatus, nil
}

// IsRegistered checks if the identity is registered or not
func (registry *contractRegistry) IsRegistered(id identity.Identity) (bool, error) {
	status, err := registry.getStatus(id)
	if err != nil {
		if err == ErrNotFound {
			// TODO: this might need to be checked on BC then
			return false, nil
		}
		return false, errors.Wrap(err, "could not check status in local db")
	}

	switch status.RegistrationStatus {
	case RegisteredProvider, RegisteredConsumer, Promoting:
		return true, nil
	default:
		return false, nil
	}
}

func (registry *contractRegistry) getStatus(identity identity.Identity) (StoredRegistrationStatus, error) {
	return registry.storage.Get(identity)
}

func (registry *contractRegistry) handleRegistrationEvent(ev IdentityRegistrationRequest) {
	status, err := registry.getStatus(identity.FromAddress(ev.Identity))
	if err != nil && err != ErrNotFound {
		log.Error().Err(err).Msg("could not get status from local db")
		return
	}

	s := InProgress
	if status.RegistrationStatus == RegisteredConsumer {
		if ev.Stake > 0 {
			s = Promoting
		} else {
			log.Info().Msgf("consumer %q already registered, skipping", ev.Identity)
			return
		}
	}

	err = registry.storeStatus(StoredRegistrationStatus{}.FromEvent(s, ev))
	if err != nil {
		log.Error().Err(err).Stack().Msg("Could not store registration status")
	}

	registry.subscribeToRegistrationEvent(identity.FromAddress(ev.Identity))
}

func (registry *contractRegistry) subscribeToRegistrationEvent(identity identity.Identity) {
	unsub := make(chan bool)

	accountantIdentities := []common.Address{
		registry.accountantAddress,
	}

	userIdentities := []common.Address{
		common.HexToAddress(identity.Address),
	}

	filterOps := &bind.WatchOpts{
		Context: context.Background(),
	}

	go func() {
		log.Info().Msgf("Waiting on identities %s accountant %s", userIdentities[0].Hex(), accountantIdentities[0].Hex())
		sink := make(chan *bindings.RegistryRegisteredIdentity)
		subscription, err := registry.filterer.WatchRegisteredIdentity(filterOps, sink, userIdentities, accountantIdentities)
		if err != nil {
			registry.publisher.Publish(RegistrationEventTopic, RegistrationEventPayload{
				ID:     identity,
				Status: RegistrationError,
			})
			log.Error().Err(err).Msg("could not register to identity events")

			updateErr := registry.storage.UpdateStatus(identity, RegistrationError)
			if updateErr != nil {
				log.Error().Err(updateErr).Msg("could not store registration status")
			}
			return
		}
		defer subscription.Unsubscribe()

		// TODO: add appropriate timeout
		select {
		case <-unsub:
			registry.publisher.Publish(RegistrationEventTopic, RegistrationEventPayload{
				ID:     identity,
				Status: RegistrationError,
			})
			err := registry.storage.UpdateStatus(identity, RegistrationError)
			if err != nil {
				log.Error().Err(err).Msg("could not store registration status")
			}
		case <-sink:
			s, err := registry.getStatus(identity)
			if err != nil {
				log.Error().Err(err).Msg("could not store registration status")
			}

			status := RegisteredConsumer
			if s.RegistrationStatus == Promoting {
				status = RegisteredProvider
			}

			registry.publisher.Publish(RegistrationEventTopic, RegistrationEventPayload{
				ID:     identity,
				Status: status,
			})

			err = registry.storage.UpdateStatus(identity, status)
			if err != nil {
				log.Error().Err(err).Msg("could not store registration status")
			}
		case err := <-subscription.Err():
			registry.publisher.Publish(RegistrationEventTopic, RegistrationEventPayload{
				ID:     identity,
				Status: RegistrationError,
			})
			if err != nil {
				log.Error().Err(err).Msg("Subscription error")
			}
			updateErr := registry.storage.UpdateStatus(identity, RegistrationError)
			if updateErr != nil {
				log.Error().Err(updateErr).Msg("could not store registration status")
			}
		}
	}()
	return
}

type contractRegistry struct {
	accountantSession          *bindings.AccountantImplementationCallerSession
	contractSession            *bindings.RegistryCallerSession
	filterer                   *bindings.RegistryFilterer
	accountantAddress          common.Address
	storage                    registryStorage
	cacheLock                  sync.Mutex
	stop                       chan struct{}
	once                       sync.Once
	registrationCompletionLock sync.Mutex
	publisher                  eventbus.Publisher
}

func (registry *contractRegistry) handleStop() {
	registry.once.Do(func() {
		log.Info().Msg("Stopping registry...")
		close(registry.stop)
	})
}

func (registry *contractRegistry) handleStart() {
	log.Info().Msg("starting registry...")
	err := registry.loadInitialState()
	if err != nil {
		log.Error().Err(err).Msg("could not start registry")
	}
}

const month = time.Hour * 24 * 7 * 4

func (registry *contractRegistry) loadInitialState() error {
	entries, err := registry.storage.GetAll()
	if err != nil {
		return errors.Wrap(err, "could not fetch previous registrations")
	}

	for i := range entries {
		if time.Now().UTC().Sub(entries[i].UpdatedAt) > month {
			log.Info().Msgf("Skipping identity %q as it has not been updated recently", entries[i].Identity)
			continue
		}
		switch entries[i].RegistrationStatus {
		case RegistrationError, InProgress, Promoting, RegisteredConsumer:
			err := registry.handleUnregisteredIdentityInitialLoad(entries[i].Identity)
			if err != nil {
				return errors.Wrapf(err, "could not check %q registration status", entries[i].Identity)
			}
		default:
			log.Info().Msgf("Identity %q already registered, skipping", entries[i].Identity)
		}
	}
	return nil
}

func (registry *contractRegistry) getProviderChannelAddressBytes(providerIdentity identity.Identity) ([32]byte, error) {
	providerAddress := providerIdentity.ToCommonAddress()
	addressBytes := [32]byte{}

	addr, err := crypto.GenerateProviderChannelID(providerAddress.Hex(), registry.accountantAddress.Hex())
	if err != nil {
		return addressBytes, errors.Wrap(err, "could not generate channel address")
	}

	padded := crypto.Pad(common.FromHex(addr), 32)
	copy(addressBytes[:], padded)

	return addressBytes, nil
}

func (registry *contractRegistry) handleUnregisteredIdentityInitialLoad(id identity.Identity) error {
	registered, err := registry.isRegisteredInBC(id)
	if err != nil {
		return errors.Wrap(err, "could not check status on blockchain")
	}

	switch registered {
	case RegisteredConsumer, RegisteredProvider:
		err := registry.storage.UpdateStatus(id, registered)
		if err != nil {
			return errors.Wrap(err, "could not store registration status on local db")
		}
	default:
		registry.subscribeToRegistrationEvent(id)
	}
	return nil
}

func (registry *contractRegistry) isRegisteredInBC(id identity.Identity) (RegistrationStatus, error) {
	registered, err := registry.contractSession.IsRegistered(
		common.HexToAddress(id.Address),
	)
	if err != nil {
		return RegistrationError, errors.Wrap(err, "could not check registration status in bc")
	}

	if !registered {
		return Unregistered, nil
	}

	providerAddressBytes, err := registry.getProviderChannelAddressBytes(id)
	if err != nil {
		return RegistrationError, errors.Wrap(err, "could not get provider channel address")
	}
	log.Debug().Msgf("providerAddressBytes %v", providerAddressBytes)

	providerChannel, err := registry.accountantSession.Channels(providerAddressBytes)
	if err != nil {
		return RegistrationError, errors.Wrap(err, "could not get provider channel")
	}
	log.Debug().Msgf("provider address %v b %v l %v s %v", hex.EncodeToString(providerAddressBytes[:]), providerChannel.Balance.Uint64(), providerChannel.Loan.Uint64(), providerChannel.Settled.Uint64())

	if providerChannel.Loan.Cmp(big.NewInt(0)) == 1 {
		return RegisteredProvider, nil
	}

	return RegisteredConsumer, nil
}
