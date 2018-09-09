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
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
)

// ProposalStatus describes stage of proposal registration
type ProposalStatus int

// Proposal registration stages
const (
	IdentityUnregistered ProposalStatus = iota
	WaitingForRegistration
	IdentityRegisterFailed
	RegisterProposal
	PingProposal
	UnregisterProposal
	UnregisterProposalFailed
	ProposalUnregistered
	StatusUndefined
)

const logPrefix = "[discovery] "

// Start launches discovery service
func (d *Discovery) Start(ownIdentity identity.Identity, proposal dto_discovery.ServiceProposal) {
	d.RLock()
	defer d.RUnlock()

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
		case event := <-d.proposalStatusChan:
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
	log.Info(logPrefix, "stopping discovery loop..")
	d.RLock()
	if d.status == WaitingForRegistration {
		d.unsubscribe()
	}
	d.RUnlock()

	d.RLock()
	if d.status == RegisterProposal || d.status == PingProposal {
		d.RUnlock()
		d.sendEvent(UnregisterProposal)
		return
	}
	d.RUnlock()
}

func (d *Discovery) registerIdentity() {
	registerEventChan, unsubscribe := d.identityRegistry.SubscribeToRegistrationEvent(d.ownIdentity)
	d.unsubscribe = unsubscribe
	d.sendEvent(WaitingForRegistration)
	go func() {
		registerEvent := <-registerEventChan
		switch registerEvent {
		case identity_registry.Registered:
			log.Info(logPrefix, "identity registered, proceeding with proposal registration")
			d.sendEvent(RegisterProposal)
		case identity_registry.Cancelled:
			log.Info(logPrefix, "cancelled identity registration")
			d.sendEvent(IdentityRegisterFailed)
		}
	}()
}

func (d *Discovery) registerProposal() {
	err := d.mysteriumClient.RegisterProposal(d.proposal, d.signer)
	if err != nil {
		log.Errorf(logPrefix, "Failed to register proposal: %v, retrying after 1 min.", err)
		time.Sleep(1 * time.Minute)
		d.sendEvent(RegisterProposal)
	}
	d.sendEvent(PingProposal)
}

func (d *Discovery) pingProposal() {
	time.Sleep(1 * time.Minute)
	err := d.mysteriumClient.PingProposal(d.proposal, d.signer)
	if err != nil {
		log.Error(logPrefix, "Failed to ping proposal: ", err)
	}
	d.sendEvent(PingProposal)
}

func (d *Discovery) unregisterProposal() {
	err := d.mysteriumClient.UnregisterProposal(d.proposal, d.signer)
	if err != nil {
		log.Error(logPrefix, "Failed to unregister proposal: ", err)
		d.sendEvent(UnregisterProposalFailed)
	}
	log.Info(logPrefix, "Proposal unregistered")
	d.sendEvent(ProposalUnregistered)
}

func (d *Discovery) checkRegistration() {
	// check if node's identity is registered
	registered, err := d.identityRegistry.IsRegistered(d.ownIdentity)
	if err != nil {
		d.sendEvent(IdentityRegisterFailed)
		return
	}

	if !registered {
		// if not registered - wait indefinitely for identity registration event
		registrationData, err := d.identityRegistration.ProvideRegistrationData(d.ownIdentity)
		if err != nil {
			d.sendEvent(IdentityRegisterFailed)
			return
		}
		identity_registry.PrintRegistrationData(registrationData)
		log.Infof("%s identity %s not registered, delaying proposal registration until identity is registered", logPrefix, d.ownIdentity.Address)
		d.sendEvent(IdentityUnregistered)
		return
	}
	d.sendEvent(RegisterProposal)
}

func (d *Discovery) sendEvent(event ProposalStatus) {
	d.Lock()
	defer d.Unlock()
	d.status = event
	go func() {
		d.proposalStatusChan <- event
	}()
}
