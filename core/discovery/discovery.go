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

	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
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

// ProposalRegistry defines methods for proposal lifecycle - registration, keeping up to date, removal
type ProposalRegistry interface {
	RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error
	PingProposal(proposal market.ServiceProposal, signer identity.Signer) error
	UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error
}

// Discovery structure holds discovery service state
type Discovery struct {
	identityRegistry     identity_registry.IdentityRegistry
	ownIdentity          identity.Identity
	identityRegistration identity_registry.RegistrationDataProvider
	proposalRegistry     ProposalRegistry
	signerCreate         identity.SignerFactory
	signer               identity.Signer
	proposal             market.ServiceProposal

	statusChan                  chan Status
	status                      Status
	proposalAnnouncementStopped *sync.WaitGroup
	unsubscribe                 func()
	stop                        func()

	mu sync.RWMutex
}

// NewService creates new discovery service
func NewService(
	identityRegistry identity_registry.IdentityRegistry,
	identityRegistration identity_registry.RegistrationDataProvider,
	proposalRegistry ProposalRegistry,
	signerCreate identity.SignerFactory,
) *Discovery {
	return &Discovery{
		identityRegistry:            identityRegistry,
		identityRegistration:        identityRegistration,
		proposalRegistry:            proposalRegistry,
		signerCreate:                signerCreate,
		statusChan:                  make(chan Status),
		status:                      StatusUndefined,
		proposalAnnouncementStopped: &sync.WaitGroup{},
		unsubscribe:                 func() {},
		stop:                        func() {},
	}
}

// Start launches discovery service
func (d *Discovery) Start(ownIdentity identity.Identity, proposal market.ServiceProposal) {
	log.Info("starting discovery...")
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.ownIdentity = ownIdentity
	d.signer = d.signerCreate(ownIdentity)
	d.proposal = proposal

	stopLoop := make(chan bool)
	d.stop = func() {
		// cancel (stop) discovery loop
		stopLoop <- true
	}

	d.proposalAnnouncementStopped.Add(1)

	go d.checkRegistration()

	go d.mainDiscoveryLoop(stopLoop)
}

// Wait wait for proposal announcements to stop / unregister
func (d *Discovery) Wait() {
	d.proposalAnnouncementStopped.Wait()
}

// Stop stops discovery loop
func (d *Discovery) Stop() {
	d.stop()
}

func (d *Discovery) mainDiscoveryLoop(stopLoop chan bool) {

	for {
		select {
		case <-stopLoop:
			d.stopLoop()
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
				d.proposalAnnouncementStopped.Done()
				return
			}
		}
	}
}

func (d *Discovery) stopLoop() {
	log.Info("stopping discovery loop..")
	d.mu.RLock()
	if d.status == WaitingForRegistration {
		d.mu.RUnlock()
		d.unsubscribe()
		d.mu.RLock()
	}

	if d.status == RegisterProposal || d.status == PingProposal {
		d.mu.RUnlock()
		d.changeStatus(UnregisterProposal)
		return
	}
	d.mu.RUnlock()
}

func (d *Discovery) registerIdentity() {
	registerEventChan, unsubscribe := d.identityRegistry.SubscribeToRegistrationEvent(d.ownIdentity)
	d.unsubscribe = unsubscribe
	d.changeStatus(WaitingForRegistration)
	go func() {
		registerEvent := <-registerEventChan
		switch registerEvent {
		case identity_registry.Registered:
			log.Info("identity registered, proceeding with proposal registration")
			d.changeStatus(RegisterProposal)
		case identity_registry.Cancelled:
			log.Info("cancelled identity registration")
			d.changeStatus(IdentityRegisterFailed)
		}
	}()
}

func (d *Discovery) registerProposal() {
	err := d.proposalRegistry.RegisterProposal(d.proposal, d.signer)
	if err != nil {
		log.Errorf("failed to register proposal, retrying after 1 min. %s", err.Error())
		time.Sleep(1 * time.Minute)
		d.changeStatus(RegisterProposal)
		return
	}
	d.changeStatus(PingProposal)
}

func (d *Discovery) pingProposal() {
	time.Sleep(1 * time.Minute)
	err := d.proposalRegistry.PingProposal(d.proposal, d.signer)
	if err != nil {
		log.Error("failed to ping proposal: ", err)
	}
	d.changeStatus(PingProposal)
}

func (d *Discovery) unregisterProposal() {
	err := d.proposalRegistry.UnregisterProposal(d.proposal, d.signer)
	if err != nil {
		log.Error("failed to unregister proposal: ", err)
		d.changeStatus(UnregisterProposalFailed)
	}
	log.Info("proposal unregistered")
	d.changeStatus(ProposalUnregistered)
}

func (d *Discovery) checkRegistration() {
	// check if node's identity is registered
	registered, err := d.identityRegistry.IsRegistered(d.ownIdentity)
	if err != nil {
		log.Error("checking identity registration failed: ", err)
		d.changeStatus(IdentityRegisterFailed)
		return
	}

	if !registered {
		// if not registered - wait indefinitely for identity registration event
		registrationData, err := d.identityRegistration.ProvideRegistrationData(d.ownIdentity)
		if err != nil {
			log.Error("fetching identity registration data failed: ", err)
			d.changeStatus(IdentityRegisterFailed)
			return
		}
		identity_registry.PrintRegistrationData(registrationData)
		log.Infof("identity %s not registered, delaying proposal registration until identity is registered", d.ownIdentity.Address)
		d.changeStatus(IdentityUnregistered)
		return
	}
	d.changeStatus(RegisterProposal)
}

func (d *Discovery) changeStatus(status Status) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.status = status

	go func() {
		d.statusChan <- status
	}()
}
