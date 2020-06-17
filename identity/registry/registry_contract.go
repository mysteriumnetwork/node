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
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/bindings"
	paymentClient "github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type registryStorage interface {
	Store(status StoredRegistrationStatus) error
	Get(identity identity.Identity) (StoredRegistrationStatus, error)
	GetAll() ([]StoredRegistrationStatus, error)
}

type contractRegistry struct {
	hermesAddress   common.Address
	storage         registryStorage
	stop            chan struct{}
	once            sync.Once
	publisher       eventbus.Publisher
	lock            sync.Mutex
	ethC            *paymentClient.ReconnectableEthClient
	registryAddress common.Address
}

// NewIdentityRegistryContract creates identity registry service which uses blockchain for information
func NewIdentityRegistryContract(ethClient *paymentClient.ReconnectableEthClient, registryAddress, hermesAddress common.Address, registryStorage registryStorage, publisher eventbus.Publisher) (*contractRegistry, error) {
	log.Info().Msgf("Using registryAddress %v hermesAddress %v", registryAddress.Hex(), hermesAddress.Hex())
	return &contractRegistry{
		hermesAddress:   hermesAddress,
		storage:         registryStorage,
		stop:            make(chan struct{}),
		publisher:       publisher,
		ethC:            ethClient,
		registryAddress: registryAddress,
	}, nil
}

// Subscribe subscribes the contract registry to relevant events
func (registry *contractRegistry) Subscribe(eb eventbus.Subscriber) error {
	err := eb.SubscribeAsync(event.AppTopicNode, registry.handleNodeEvent)
	if err != nil {
		return err
	}
	err = eb.SubscribeAsync(AppTopicEthereumClientReconnected, registry.handleEtherClientReconnect)
	if err != nil {
		return err
	}
	return eb.Subscribe(AppTopicTransactorRegistration, registry.handleRegistrationEvent)
}

// GetRegistrationStatus returns the registration status of the provided identity
func (registry *contractRegistry) GetRegistrationStatus(id identity.Identity) (RegistrationStatus, error) {
	status, err := registry.storage.Get(id)
	if err == nil {
		return status.RegistrationStatus, nil
	}

	if err != ErrNotFound {
		return Unregistered, errors.Wrap(err, "could not check status in local db")
	}

	statusBC, err := registry.isRegisteredInBC(id)
	if err != nil {
		return Unregistered, errors.Wrap(err, "could not check identity registration status on blockchain")
	}
	err = registry.storage.Store(StoredRegistrationStatus{
		Identity:           id,
		RegistrationStatus: statusBC,
	})
	return statusBC, errors.Wrap(err, "could not store registration status")
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

func (registry *contractRegistry) handleRegistrationEvent(ev IdentityRegistrationRequest) {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	status, err := registry.storage.Get(identity.FromAddress(ev.Identity))
	if err != nil && err != ErrNotFound {
		log.Error().Err(err).Msg("Could not get status from local db")
		return
	}

	if err == ErrNotFound {
		status = StoredRegistrationStatus{}.FromEvent(Unregistered, ev)
		err := registry.storage.Store(status)
		if err != nil {
			log.Error().Err(err).Stack().Msg("Could not store registration status")
			return
		}
	}

	if status.RegistrationStatus == Registered {
		log.Info().Msgf("Identity %q already registered, skipping", ev.Identity)
		return
	}

	s := InProgress

	ID := identity.FromAddress(ev.Identity)

	go registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
		ID:     ID,
		Status: s,
	})
	err = registry.storage.Store(StoredRegistrationStatus{
		Identity:           ID,
		RegistrationStatus: s,
	})
	if err != nil {
		log.Error().Err(err).Stack().Msg("Could not store registration status")
	}

	registry.subscribeToRegistrationEvent(ID)
}

func (registry *contractRegistry) subscribeToRegistrationEvent(identity identity.Identity) {
	hermesIdentities := []common.Address{
		registry.hermesAddress,
	}

	userIdentities := []common.Address{
		common.HexToAddress(identity.Address),
	}

	filterOps := &bind.WatchOpts{
		Context: context.Background(),
	}

	go func() {
		log.Info().Msgf("Waiting on identities %s hermes %s", userIdentities[0].Hex(), hermesIdentities[0].Hex())
		sink := make(chan *bindings.RegistryRegisteredIdentity)

		filterer, err := bindings.NewRegistryFilterer(registry.registryAddress, registry.ethC.Client())
		if err != nil {
			log.Error().Err(err).Msg("could not create registry filterer")
			return
		}

		subscription, err := filterer.WatchRegisteredIdentity(filterOps, sink, userIdentities, hermesIdentities)
		if err != nil {
			registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
				ID:     identity,
				Status: RegistrationError,
			})
			log.Error().Err(err).Msg("Could not register to identity events")

			updateErr := registry.storage.Store(StoredRegistrationStatus{
				Identity:           identity,
				RegistrationStatus: RegistrationError,
			})
			if updateErr != nil {
				log.Error().Err(updateErr).Msg("Could not store registration status")
			}
			return
		}
		defer subscription.Unsubscribe()

		select {
		case <-registry.stop:
			return
		case <-sink:
			log.Info().Msgf("Received registration event for %v", identity)
			_, err := registry.storage.Get(identity)
			if err != nil {
				log.Error().Err(err).Msg("Could not store registration status")
			}

			status := Registered

			log.Debug().Msgf("Sending registration success event for %v", identity)
			registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
				ID:     identity,
				Status: status,
			})

			err = registry.storage.Store(StoredRegistrationStatus{
				Identity:           identity,
				RegistrationStatus: status,
			})
			if err != nil {
				log.Error().Err(err).Msg("Could not store registration status")
			}
		case err := <-subscription.Err():
			if err == nil {
				return
			}

			log.Error().Err(err).Msg("Subscription error")
			registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
				ID:     identity,
				Status: RegistrationError,
			})
			updateErr := registry.storage.Store(StoredRegistrationStatus{
				Identity:           identity,
				RegistrationStatus: RegistrationError,
			})
			if updateErr != nil {
				log.Error().Err(updateErr).Msg("could not store registration status")
			}
		}
	}()
}

func (registry *contractRegistry) handleStop() {
	registry.once.Do(func() {
		log.Info().Msg("Stopping registry...")
		close(registry.stop)
	})
}

func (registry *contractRegistry) handleStart() {
	log.Info().Msg("Starting registry...")
	err := registry.loadInitialState()
	if err != nil {
		log.Error().Err(err).Msg("Could not start registry")
	}
}

const month = time.Hour * 24 * 7 * 4

func (registry *contractRegistry) loadInitialState() error {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	log.Debug().Msg("Loading initial state")
	entries, err := registry.storage.GetAll()
	if err != nil {
		return errors.Wrap(err, "Could not fetch previous registrations")
	}

	for i := range entries {
		if time.Now().UTC().Sub(entries[i].UpdatedAt) > month {
			log.Debug().Msgf("Skipping identity %q as it has not been updated recently", entries[i].Identity)
			continue
		}
		switch entries[i].RegistrationStatus {
		case RegistrationError, InProgress:
			err := registry.handleUnregisteredIdentityInitialLoad(entries[i].Identity)
			if err != nil {
				return errors.Wrapf(err, "could not check %q registration status", entries[i].Identity)
			}
		default:
			log.Debug().Msgf("Identity %q already registered, skipping", entries[i].Identity)
		}
	}
	return nil
}

func (registry *contractRegistry) getProviderChannelAddressBytes(providerIdentity identity.Identity) ([32]byte, error) {
	providerAddress := providerIdentity.ToCommonAddress()
	addressBytes := [32]byte{}

	addr, err := crypto.GenerateProviderChannelID(providerAddress.Hex(), registry.hermesAddress.Hex())
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
	case Registered:
		err := registry.storage.Store(StoredRegistrationStatus{
			Identity:           id,
			RegistrationStatus: registered,
		})
		if err != nil {
			return errors.Wrap(err, "could not store registration status on local db")
		}
	default:
		registry.subscribeToRegistrationEvent(id)
	}
	return nil
}

func (registry *contractRegistry) isRegisteredInBC(id identity.Identity) (RegistrationStatus, error) {
	contract, err := bindings.NewRegistryCaller(registry.registryAddress, registry.ethC.Client())
	if err != nil {
		return RegistrationError, fmt.Errorf("could not get registry caller %w", err)
	}

	contractSession := &bindings.RegistryCallerSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}

	hermesContract, err := bindings.NewHermesImplementationCaller(registry.hermesAddress, registry.ethC.Client())
	if err != nil {
		return RegistrationError, fmt.Errorf("could not get hermes implementation caller %w", err)
	}

	hermesSession := &bindings.HermesImplementationCallerSession{
		Contract: hermesContract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}

	registered, err := contractSession.IsRegistered(
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

	_, err = hermesSession.Channels(providerAddressBytes)
	if err != nil {
		return RegistrationError, errors.Wrap(err, "could not get provider channel")
	}

	return Registered, nil
}

// AppTopicEthereumClientReconnected indicates that the ethereum client has reconnected.
var AppTopicEthereumClientReconnected = "ether-client-reconnect"

func (registry *contractRegistry) handleEtherClientReconnect(_ interface{}) {
	err := registry.loadInitialState()
	if err != nil {
		log.Error().Err(err).Msg("could not resubscribe to identity status changes")
	}
}
