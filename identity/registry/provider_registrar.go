/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
	"bytes"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type registrationStatusChecker interface {
	GetRegistrationStatus(chainID int64, id identity.Identity) (RegistrationStatus, error)
}

type txer interface {
	CheckIfRegistrationBountyEligible(identity identity.Identity) (bool, error)
	GetFreeProviderRegistrationEligibility() (bool, error)
	RegisterProvider(id string, stake, fee *big.Int, beneficiary string, chainID int64) error
}

type multiChainAddressKeeper interface {
	GetRegistryAddress(chainID int64) (common.Address, error)
	GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error)
}

type bc interface {
	GetBeneficiary(chainID int64, registryAddress, identity common.Address) (common.Address, error)
}

// HermesCallerFactory allows tu produce hermes clients

// ProviderRegistrar is responsible for registering a provider once a service is started.
type ProviderRegistrar struct {
	registrationStatusChecker registrationStatusChecker
	txer                      txer
	multiChainAddressKeeper   multiChainAddressKeeper
	bc                        bc
	once                      sync.Once
	stopChan                  chan struct{}
	queue                     chan queuedEvent
	registeredIdentities      map[string]struct{}

	cfg ProviderRegistrarConfig
}

type queuedEvent struct {
	event   servicestate.AppEventServiceStatus
	retries int
}

// ProviderRegistrarConfig represents all things configurable for the provider registrar
type ProviderRegistrarConfig struct {
	MaxRetries          int
	DelayBetweenRetries time.Duration
}

// NewProviderRegistrar creates a new instance of provider registrar
func NewProviderRegistrar(
	transactor txer,
	registrationStatusChecker registrationStatusChecker,
	multiChainAddressKeeper multiChainAddressKeeper,
	bc bc,
	prc ProviderRegistrarConfig,
) *ProviderRegistrar {
	return &ProviderRegistrar{
		stopChan:                  make(chan struct{}),
		registrationStatusChecker: registrationStatusChecker,
		queue:                     make(chan queuedEvent),
		registeredIdentities:      make(map[string]struct{}),
		cfg:                       prc,
		txer:                      transactor,
		multiChainAddressKeeper:   multiChainAddressKeeper,
		bc:                        bc,
	}
}

// Subscribe subscribes the provider registrar to service state change events
func (pr *ProviderRegistrar) Subscribe(eb eventbus.EventBus) error {
	err := eb.SubscribeAsync(event.AppTopicNode, pr.handleNodeStartupEvents)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to node events")
	}
	return eb.SubscribeAsync(servicestate.AppTopicServiceStatus, pr.consumeServiceEvent)
}

func (pr *ProviderRegistrar) handleNodeStartupEvents(e event.Payload) {
	if e.Status == event.StatusStarted {
		err := pr.start()
		if err != nil {
			log.Error().Err(err).Stack().Msgf("Fatal error for provider identity registrar. Identity will not be registered. Please restart your node.")
		}
		return
	}
	if e.Status == event.StatusStopped {
		pr.stop()
		return
	}
}

func (pr *ProviderRegistrar) consumeServiceEvent(event servicestate.AppEventServiceStatus) {
	pr.queue <- queuedEvent{
		event:   event,
		retries: 0,
	}
}

func (pr *ProviderRegistrar) needsHandling(qe queuedEvent) bool {
	if qe.event.Status != string(servicestate.Running) {
		log.Debug().Msgf("Received %q service event, ignoring", qe.event.Status)
		return false
	}

	if _, ok := pr.registeredIdentities[qe.event.ProviderID]; ok {
		log.Info().Msgf("Provider %q already marked as registered, skipping", qe.event.ProviderID)
		return false
	}

	return true
}

func (pr *ProviderRegistrar) handleEventWithRetries(qe queuedEvent) error {
	err := pr.handleEvent(qe)
	if err == nil {
		return nil
	}
	if qe.retries < pr.cfg.MaxRetries {
		log.Error().Err(err).Stack().Msgf("Could not complete registration for provider %q. Will retry. Retry %v of %v", qe.event.ProviderID, qe.retries, pr.cfg.MaxRetries)
		qe.retries++
		go pr.delayedRequeue(qe)
		return nil
	}

	return errors.Wrap(err, "max attempts reached for provider registration")
}

func (pr *ProviderRegistrar) delayedRequeue(qe queuedEvent) {
	select {
	case <-pr.stopChan:
		return
	case <-time.After(pr.cfg.DelayBetweenRetries):
		pr.queue <- qe
	}
}

func (pr *ProviderRegistrar) l2chainID() int64 {
	return config.GetInt64(config.FlagChain2ChainID)
}

func (pr *ProviderRegistrar) l1chainID() int64 {
	return config.GetInt64(config.FlagChain1ChainID)
}

func (pr *ProviderRegistrar) chainID() int64 {
	return config.GetInt64(config.FlagChainID)
}

func (pr *ProviderRegistrar) handleEvent(qe queuedEvent) error {
	registered, err := pr.registrationStatusChecker.GetRegistrationStatus(pr.chainID(), identity.FromAddress(qe.event.ProviderID))
	if err != nil {
		return errors.Wrap(err, "could not check registration status on BC")
	}

	switch registered {
	case Registered:
		log.Info().Msgf("Provider %q already registered on bc, skipping", qe.event.ProviderID)
		pr.registeredIdentities[qe.event.ProviderID] = struct{}{}
		return nil
	default:
		log.Info().Msgf("Provider %q not registered on BC, will check if elgible for auto-registration", qe.event.ProviderID)
		return pr.registerIdentityIfEligible(qe)
	}
}

func (pr *ProviderRegistrar) registerIdentityIfEligible(qe queuedEvent) error {
	id := identity.FromAddress(qe.event.ProviderID)

	eligible, err := pr.txer.CheckIfRegistrationBountyEligible(id)
	if err != nil {
		log.Error().Err(err).Msgf("eligibility for registration check failed for %q", id.Address)
		return errors.Wrap(err, "could not check eligibility for auto-registration")
	}

	if eligible {
		return pr.registerIdentity(qe, id)
	}

	eligible, err = pr.txer.GetFreeProviderRegistrationEligibility()
	if err != nil {
		log.Error().Err(err).Msgf("tried to auto-registration eligibility, but failed: %q", id.Address)
		return errors.Wrap(err, "could not check auto-registration eligibility")
	}
	if eligible {
		return pr.registerIdentity(qe, id)
	}

	log.Info().Msgf("provider %q not elgible for auto-registration", id.Address)
	return nil
}

func (pr *ProviderRegistrar) registerIdentity(qe queuedEvent, id identity.Identity) error {
	if pr.chainID() != pr.l2chainID() {
		return errors.New("registration failed, cannot register on l1 chain")
	}

	// If chain is l2 (matic) beneficiary should be set to channel address
	settleBeneficiary, err := pr.multiChainAddressKeeper.GetChannelAddress(pr.chainID(), id)
	if err != nil {
		log.Error().Err(err).Msg("Registration failed for could not generate channel address")
		return err
	}

	err = pr.txer.RegisterProvider(qe.event.ProviderID, big.NewInt(0), nil, settleBeneficiary.Hex(), pr.chainID())
	if err != nil {
		log.Error().Err(err).Msgf("Registration failed for provider %q", qe.event.ProviderID)
		return errors.Wrap(err, "could not register identity on BC")
	}

	pr.registeredIdentities[qe.event.ProviderID] = struct{}{}
	log.Info().Msgf("Registration success for provider %q", id.Address)
	return nil
}

var newRegistryAddress = common.HexToAddress("0xDFAB03C9fbDbef66dA105B88776B35bfd7743D39")

func isZeroAddress(in common.Address) bool {
	var zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
	return bytes.EqualFold(in.Bytes(), zeroAddress.Bytes())
}

// start starts the provider registrar
func (pr *ProviderRegistrar) start() error {
	log.Info().Msg("Starting provider registrar")
	for {
		select {
		case <-pr.stopChan:
			return nil
		case event := <-pr.queue:
			if !pr.needsHandling(event) {
				break
			}

			err := pr.handleEventWithRetries(event)
			if err != nil {
				return err
			}
		}
	}
}

func (pr *ProviderRegistrar) stop() {
	pr.once.Do(func() {
		log.Info().Msg("Stopping provider registrar")
		close(pr.stopChan)
	})
}
