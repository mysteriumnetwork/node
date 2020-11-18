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
	RegisterIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error
	CheckIfRegistrationBountyEligible(identity identity.Identity) (bool, error)
}

// ProviderRegistrar is responsible for registering a provider once a service is started.
type ProviderRegistrar struct {
	registrationStatusChecker registrationStatusChecker
	txer                      txer
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
	IsTestnet2          bool
	MaxRetries          int
	Stake               *big.Int
	DelayBetweenRetries time.Duration
	HermesAddress       common.Address
	RegistryAddress     common.Address
}

// NewProviderRegistrar creates a new instance of provider registrar
func NewProviderRegistrar(transactor txer, registrationStatusChecker registrationStatusChecker, prc ProviderRegistrarConfig) *ProviderRegistrar {
	return &ProviderRegistrar{
		stopChan:                  make(chan struct{}),
		registrationStatusChecker: registrationStatusChecker,
		queue:                     make(chan queuedEvent),
		registeredIdentities:      make(map[string]struct{}),
		cfg:                       prc,
		txer:                      transactor,
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
		return pr.registerIdentity(qe)
	}
}

func (pr *ProviderRegistrar) registerIdentity(qe queuedEvent) error {
	if !pr.cfg.IsTestnet2 {
		eligible, err := pr.txer.CheckIfRegistrationBountyEligible(identity.FromAddress(qe.event.ProviderID))
		if err != nil {
			log.Error().Err(err).Msgf("eligibility for registration check failed for %q", qe.event.ProviderID)
			return errors.Wrap(err, "could not check eligibility for auto-registration")
		}

		if !eligible {
			log.Info().Msgf("provider %q not eligible for auto registration, will require manual registration", qe.event.ProviderID)
			return nil
		}
	}

	err := pr.txer.RegisterIdentity(qe.event.ProviderID, pr.cfg.Stake, big.NewInt(0), "", pr.chainID(), nil)
	if err != nil {
		log.Error().Err(err).Msgf("Registration failed for provider %q", qe.event.ProviderID)
		return errors.Wrap(err, "could not register identity on BC")
	}

	pr.registeredIdentities[qe.event.ProviderID] = struct{}{}
	log.Info().Msgf("Registration success for provider %q", qe.event.ProviderID)
	return nil
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
