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

package discovery

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
)

// Status describes stage of proposal registration
type Status int

// Proposal registration stages
const (
	IdentityUnregistered Status = iota
	WaitingForRegistration
	IdentityRegisterFailed
	RegisterProposal
	PingProposal
	UnregisterProposal
	UnregisterProposalFailed
	ProposalUnregistered
	StatusUndefined
)

// Discovery structure holds discovery service state
type Discovery struct {
	identityRegistry registry.IdentityRegistry
	ownIdentity      identity.Identity
	proposalRegistry ProposalRegistry
	proposalPingTTL  time.Duration
	signerCreate     identity.SignerFactory
	signer           identity.Signer
	proposal         func() market.ServiceProposal
	eventBus         eventbus.EventBus

	statusChan                  chan Status
	status                      Status
	proposalAnnouncementStopped *sync.WaitGroup
	stop                        chan struct{}
	once                        sync.Once

	mu sync.RWMutex
}

// NewService creates new discovery service
func NewService(
	identityRegistry registry.IdentityRegistry,
	proposalRegistry ProposalRegistry,
	proposalPingTTL time.Duration,
	signerCreate identity.SignerFactory,
	eventBus eventbus.EventBus,
) *Discovery {
	return &Discovery{
		identityRegistry:            identityRegistry,
		proposalRegistry:            proposalRegistry,
		proposalPingTTL:             proposalPingTTL,
		eventBus:                    eventBus,
		signerCreate:                signerCreate,
		statusChan:                  make(chan Status),
		status:                      StatusUndefined,
		proposalAnnouncementStopped: &sync.WaitGroup{},
		stop:                        make(chan struct{}),
	}
}

// Start launches discovery service
func (d *Discovery) Start(ownIdentity identity.Identity, proposal func() market.ServiceProposal) {
	log.Info().Msg("Starting discovery...")
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.ownIdentity = ownIdentity
	d.signer = d.signerCreate(ownIdentity)
	d.proposal = proposal

	d.proposalAnnouncementStopped.Add(1)

	go d.checkRegistration()

	go d.mainDiscoveryLoop()
}

// Wait wait for proposal announcements to stop / unregister
func (d *Discovery) Wait() {
	d.proposalAnnouncementStopped.Wait()
}

// Stop stops discovery loop
func (d *Discovery) Stop() {
	d.once.Do(func() {
		close(d.stop)
	})
}

func (d *Discovery) mainDiscoveryLoop() {
	defer d.proposalAnnouncementStopped.Done()
	for {
		select {
		case <-d.stop:
			d.stopLoop()
			d.unregisterProposal()
			return
		case event := <-d.statusChan:
			switch event {
			case IdentityUnregistered:
				d.registerIdentity()
			case RegisterProposal:
				go d.registerProposal()
			case PingProposal:
				go d.pingProposal()
			case UnregisterProposal:
				go d.unregisterProposal()
			case IdentityRegisterFailed, ProposalUnregistered, UnregisterProposalFailed:
				return
			}
		}
	}
}

func (d *Discovery) stopLoop() {
	log.Info().Msg("Stopping discovery loop...")
	d.mu.RLock()
	if d.status == WaitingForRegistration {
		d.mu.RUnlock()
		d.mu.RLock()
	}

	if d.status == RegisterProposal || d.status == PingProposal {
		d.mu.RUnlock()
		d.changeStatus(UnregisterProposal)
		return
	}
	d.mu.RUnlock()
}

func (d *Discovery) handleRegistrationEvent(rep registry.AppEventIdentityRegistration) {
	log.Debug().Msgf("Registration event received for %v", rep.ID.Address)
	if rep.ID.Address != d.ownIdentity.Address {
		log.Debug().Msgf("Identity mismatch for registration. Expected %v got %v", d.ownIdentity.Address, rep.ID.Address)
		return
	}

	switch rep.Status {
	case registry.Registered:
		log.Info().Msg("Identity registered, proceeding with proposal registration")
		d.changeStatus(RegisterProposal)
	case registry.RegistrationError:
		log.Info().Msg("Cancelled identity registration")
		d.changeStatus(IdentityRegisterFailed)
	default:
		log.Info().Msgf("Received status %v ignoring", rep.Status)
	}
}

func (d *Discovery) registerIdentity() {
	log.Info().Msg("Waiting for registration success event")
	d.eventBus.Subscribe(registry.AppTopicIdentityRegistration, d.handleRegistrationEvent)
	d.changeStatus(WaitingForRegistration)
}

func (d *Discovery) registerProposal() {
	proposal := d.proposal()
	err := d.proposalRegistry.RegisterProposal(proposal, d.signer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to register proposal, retrying after 1 min")
		select {
		case <-d.stop:
			return
		case <-time.After(1 * time.Minute):
			d.changeStatus(RegisterProposal)
			return
		}
	}
	d.eventBus.Publish(AppTopicProposalAnnounce, proposal)
	d.changeStatus(PingProposal)
}

func (d *Discovery) pingProposal() {
	select {
	case <-d.stop:
		return
	case <-time.After(d.proposalPingTTL):
		proposal := d.proposal()
		err := d.proposalRegistry.PingProposal(proposal, d.signer)
		if err != nil {
			log.Error().Err(err).Msg("Failed to ping proposal")
		}
		d.eventBus.Publish(AppTopicProposalAnnounce, proposal)
		d.changeStatus(PingProposal)
	}
}

func (d *Discovery) unregisterProposal() {
	proposal := d.proposal()
	err := d.proposalRegistry.UnregisterProposal(proposal, d.signer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unregister proposal: ")
		d.changeStatus(UnregisterProposalFailed)
	}
	log.Info().Msg("Proposal unregistered")
	d.changeStatus(ProposalUnregistered)
}

func (d *Discovery) checkRegistration() {
	// check if node's identity is registered
	chainID := config.GetInt64(config.FlagChainID)
	status, err := d.identityRegistry.GetRegistrationStatus(chainID, d.ownIdentity)
	if err != nil {
		log.Error().Err(err).Msg("Checking identity registration failed")
		d.changeStatus(IdentityRegisterFailed)
		return
	}
	switch status {
	case registry.Registered:
		d.changeStatus(RegisterProposal)
	default:
		log.Info().Msgf("Identity %s not registered, delaying proposal registration until identity is registered", d.ownIdentity.Address)
		d.changeStatus(IdentityUnregistered)
		return
	}
}

func (d *Discovery) changeStatus(status Status) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status = status

	go func() {
		d.statusChan <- status
	}()
}
