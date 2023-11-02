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
	"fmt"
	"sync"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/payments/bindings"
	paymentClient "github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type registryStorage interface {
	Store(status StoredRegistrationStatus) error
	Get(chainID int64, identity identity.Identity) (StoredRegistrationStatus, error)
	GetAll() ([]StoredRegistrationStatus, error)
}

type hermesCaller interface {
	IsIdentityOffchain(chainID int64, id string) (bool, error)
}

type transactor interface {
	FetchRegistrationStatus(id string) ([]TransactorStatusResponse, error)
	GetFreeProviderRegistrationEligibility() (bool, error)
	RegisterProviderIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error
}

type contractRegistry struct {
	storage    registryStorage
	stop       chan struct{}
	once       sync.Once
	publisher  eventbus.Publisher
	lock       sync.Mutex
	ethC       paymentClient.EtherClient
	ap         AddressProvider
	hermes     hermesCaller
	transactor transactor
	manager    identity.Manager
	cfg        IdentityRegistryConfig
}

// IdentityRegistryConfig contains the configuration for registry contract.
type IdentityRegistryConfig struct {
	TransactorPollInterval time.Duration
	TransactorPollTimeout  time.Duration
}

// NewIdentityRegistryContract creates identity registry service which uses blockchain for information
func NewIdentityRegistryContract(ethClient paymentClient.EtherClient, ap AddressProvider, registryStorage registryStorage, publisher eventbus.Publisher, caller hermesCaller, transactor transactor, selector identity_selector.Handler, cfg IdentityRegistryConfig) (*contractRegistry, error) {
	return &contractRegistry{
		storage:    registryStorage,
		stop:       make(chan struct{}),
		publisher:  publisher,
		ethC:       ethClient,
		ap:         ap,
		hermes:     caller,
		transactor: transactor,
		cfg:        cfg,
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
func (registry *contractRegistry) GetRegistrationStatus(chainID int64, id identity.Identity) (RegistrationStatus, error) {
	var currentStatus RegistrationStatus
	ss, err := registry.storage.Get(chainID, id)
	switch err {
	case nil:
		currentStatus = ss.RegistrationStatus
	case ErrNotFound:
		currentStatus = Unregistered
	default:
		return Unregistered, errors.Wrap(err, "could not check status in local db")
	}

	if currentStatus == Registered {
		return currentStatus, nil
	}

	var newStatus RegistrationStatus
	newStatus, err = registry.bcRegistrationStatus(chainID, id)
	if err != nil {
		return Unregistered, errors.Wrap(err, "could not check identity registration status on blockchain")
	}

	if newStatus == Unregistered && ss.RegistrationStatus == InProgress {
		newStatus = InProgress
	}

	if newStatus == Unregistered || newStatus == InProgress {
		ok, err := registry.hermes.IsIdentityOffchain(chainID, id.Address)
		if err != nil {
			log.Err(err).Str("status", newStatus.String()).Msg("failed to contact hermes to get new registration status")
		}

		if ok && err == nil {
			log.Debug().Str("identity", id.Address).Msg("identity is offchain, considering it registered")
			newStatus = Registered
		}
	}

	err = registry.storage.Store(StoredRegistrationStatus{
		Identity:           id,
		RegistrationStatus: newStatus,
		ChainID:            chainID,
	})
	if err != nil {
		return newStatus, errors.Wrap(err, "could not store registration status")
	}

	// If current status was not registered and we are now registered
	// publish an event for that to make sure that wasn't missed.
	if currentStatus != newStatus && newStatus == Registered {
		go registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
			ID:      id,
			Status:  newStatus,
			ChainID: chainID,
		})
	}
	return newStatus, nil
}

func (registry *contractRegistry) handleNodeEvent(ev event.Payload) {
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

	status, err := registry.storage.Get(ev.ChainID, identity.FromAddress(ev.Identity))
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

	// In case we have a previous registration, force re-check the BC status
	if status.RegistrationStatus == InProgress || status.RegistrationStatus == RegistrationError {
		status, err := registry.GetRegistrationStatus(ev.ChainID, identity.FromAddress(ev.Identity))
		if err != nil {
			log.Info().Err(err).Msg("could not recheck status with bc")
		} else if status.Registered() {
			s = Registered
		}
	}

	ID := identity.FromAddress(ev.Identity)

	go registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
		ID:      ID,
		Status:  s,
		ChainID: ev.ChainID,
	})
	err = registry.storage.Store(StoredRegistrationStatus{
		Identity:           ID,
		RegistrationStatus: s,
		ChainID:            ev.ChainID,
	})
	if err != nil {
		log.Error().Err(err).Stack().Msg("Could not store registration status")
	}

	go registry.subscribeToRegistrationEventViaTransactor(ev)
}

func (registry *contractRegistry) subscribeToRegistrationEventViaTransactor(ev IdentityRegistrationRequest) {
	timeout := time.After(registry.cfg.TransactorPollTimeout)
	for {
		select {
		case <-registry.stop:
			registry.saveRegistrationStatus(ev.ChainID, ev.Identity, RegistrationError)
			return
		case <-timeout:
			log.Info().Msg("registration watch subscription timed out")
			registry.saveRegistrationStatus(ev.ChainID, ev.Identity, RegistrationError)
			return
		case <-time.After(registry.cfg.TransactorPollInterval):
			res, err := registry.transactor.FetchRegistrationStatus(ev.Identity)
			if err != nil {
				log.Warn().Err(err).Msg("could not fetch registration status from transactor")
				break
			}

			var resp *TransactorStatusResponse
			for _, v := range res {
				if v.ChainID != ev.ChainID {
					continue
				}
				resp = &v
				break
			}

			if resp == nil {
				log.Warn().Msg("no matching registration entries for chain in transactor response, will continue")
				break
			}

			switch resp.Status {
			case TransactorRegistrationEntryStatusSucceed:
				registry.resyncWithBC(ev.ChainID, ev.Identity)
				return
			case TransactorRegistrationEntryStatusFailed:
				log.Error().Msg("registration reported as failed by transactor, will check in bc just in case")
				bcStatus, err := registry.bcRegistrationStatus(ev.ChainID, identity.FromAddress(ev.Identity))
				if err != nil {
					log.Err(err).Msg("got registration failed from transactor and failed to check registration status")
					registry.saveRegistrationStatus(ev.ChainID, ev.Identity, RegistrationError)
					return
				}

				statusToWrite := Registered
				if bcStatus != Registered {
					statusToWrite = RegistrationError
				}

				registry.saveRegistrationStatus(ev.ChainID, ev.Identity, statusToWrite)
				return
			}
		}
	}
}

func (registry *contractRegistry) resyncWithBC(chainID int64, id string) {
	status, err := registry.bcRegistrationStatus(chainID, identity.FromAddress(id))
	if err != nil {
		log.Err(err).Msg("could not check registration status on chain")
		registry.saveRegistrationStatus(chainID, id, RegistrationError)
		return
	}
	registry.saveRegistrationStatus(chainID, id, status)
}

func (registry *contractRegistry) saveRegistrationStatus(chainID int64, id string, status RegistrationStatus) {
	err := registry.storage.Store(StoredRegistrationStatus{
		Identity:           identity.FromAddress(id),
		RegistrationStatus: status,
		ChainID:            chainID,
	})
	if err != nil {
		log.Error().Err(err).Msg("Could not store registration status")
	}

	registry.publisher.Publish(AppTopicIdentityRegistration, AppEventIdentityRegistration{
		ID:      identity.FromAddress(id),
		Status:  status,
		ChainID: chainID,
	})
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

func (registry *contractRegistry) loadInitialState() error {
	log.Debug().Msg("Loading initial state")
	registry.lock.Lock()
	defer registry.lock.Unlock()

	entries, err := registry.storage.GetAll()
	if err != nil {
		return errors.Wrap(err, "Could not fetch previous registrations")
	}

	for i := range entries {
		switch entries[i].RegistrationStatus {
		case RegistrationError, InProgress:
			entry := entries[i]
			err := registry.handleUnregisteredIdentityInitialLoad(entry.ChainID, entry.Identity)
			if err != nil {
				return errors.Wrapf(err, "could not check %q registration status", entries[i].Identity)
			}
		default:
			log.Debug().Msgf("Identity %q already registered, skipping", entries[i].Identity)
		}
	}

	return nil
}

func (registry *contractRegistry) getProviderChannelAddressBytes(hermesAddress common.Address, providerIdentity identity.Identity) ([32]byte, error) {
	providerAddress := providerIdentity.ToCommonAddress()
	addressBytes := [32]byte{}

	addr, err := crypto.GenerateProviderChannelID(providerAddress.Hex(), hermesAddress.Hex())
	if err != nil {
		return addressBytes, errors.Wrap(err, "could not generate channel address")
	}

	padded := crypto.Pad(common.FromHex(addr), 32)
	copy(addressBytes[:], padded)

	return addressBytes, nil
}

func (registry *contractRegistry) handleUnregisteredIdentityInitialLoad(chainID int64, id identity.Identity) error {
	status, err := registry.GetRegistrationStatus(chainID, id)
	if err != nil {
		return errors.Wrap(err, "could not check status on blockchain")
	}

	if status == Registered {
		return nil
	}

	registry.resyncWithBC(chainID, id.Address)
	return nil
}

func (registry *contractRegistry) bcRegistrationStatus(chainID int64, id identity.Identity) (RegistrationStatus, error) {
	reg, err := registry.ap.GetRegistryAddress(chainID)
	if err != nil {
		log.Error().Err(err).Msg("could not get registry address")
		return RegistrationError, err
	}

	hermes, err := registry.ap.GetActiveHermes(chainID)
	if err != nil {
		log.Error().Err(err).Msg("could not get hermes address")
		return RegistrationError, err
	}

	contract, err := bindings.NewRegistryCaller(reg, registry.ethC)
	if err != nil {
		return RegistrationError, fmt.Errorf("could not get registry caller %w", err)
	}

	contractSession := &bindings.RegistryCallerSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false, //we want to find out true registration status - not pending transactions
		},
	}

	hermesContract, err := bindings.NewHermesImplementationCaller(hermes, registry.ethC)
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

	providerAddressBytes, err := registry.getProviderChannelAddressBytes(hermes, id)
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
