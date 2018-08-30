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

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity/registry"
)

// ProposalStatus describes stage of proposal registration
type ProposalStatus int

// Proposal registration stages
const (
	IdentityUnregistered ProposalStatus = iota
	IdentityRegisterFailed
	RegisterProposal
	PingProposal
	UnregisterProposal
	UnregisterProposalFailed
	ProposalUnregistered
)

const logPrefix = "[discovery] "

// Start launches discovery service
func (d *Discovery) Start(proposalAnnouncementStopped *sync.WaitGroup) (func(), error) {
	stopLoop := make(chan bool)
	stopDiscovery := func() {
		// cancel (stop) discovery loop
		stopLoop <- true
	}

	proposalAnnouncementStopped.Add(1)

	go d.checkRegistration()

	go d.mainDiscoveryLoop(stopLoop, proposalAnnouncementStopped)

	return stopDiscovery, nil
}

func (d *Discovery) mainDiscoveryLoop(stopLoop chan bool, proposalAnnouncementStopped *sync.WaitGroup) {
	unsubscribe := func() {}

	for {
		select {
		case <-stopLoop:
			d.stopLoop(unsubscribe)
		case event := <-d.proposalStatusChan:
			switch event {
			case IdentityUnregistered:
				unsubscribe = d.registerIdentity()
			case RegisterProposal:
				go d.registerProposal()
			case PingProposal:
				go d.pingProposal()
			case UnregisterProposal:
				go d.unregisterProposal()
			case IdentityRegisterFailed, ProposalUnregistered, UnregisterProposalFailed:
				proposalAnnouncementStopped.Done()
				return
			}
		}
	}
}

func (d *Discovery) stopLoop(unsubscribe func()) {
	log.Info(logPrefix, "stopping discovery loop..")
	unsubscribe()
	if d.status == RegisterProposal || d.status == PingProposal {
		d.sendEvent(UnregisterProposal)
	}
}

func (d *Discovery) registerIdentity() func() {
	registerEventChan, unsubscribe := d.identityRegistry.SubscribeToRegistrationEvent(d.ownIdentity)

	go func() {
		registerEvent := <-registerEventChan
		switch registerEvent {
		case registry.Registered:
			log.Info(logPrefix, "identity registered, proceeding with proposal registration")
			d.sendEvent(RegisterProposal)
		case registry.Cancelled:
			log.Info(logPrefix, "cancelled identity registration")
			d.sendEvent(IdentityRegisterFailed)
		}
	}()

	return unsubscribe
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
	}

	// if not registered - wait indefinitely for identity registration event
	if !registered {
		registrationData, err := d.registrationDataProvider.ProvideRegistrationData(d.ownIdentity)
		if err != nil {
			d.sendEvent(IdentityRegisterFailed)
		}
		registry.PrintRegistrationData(registrationData)
		log.Infof("%s identity %s not registered, delaying proposal registration until identity is registered", logPrefix, d.ownIdentity.String())
		d.sendEvent(IdentityUnregistered)
	}
}

func (d *Discovery) sendEvent(event ProposalStatus) {
	d.status = event
	go func() {
		d.proposalStatusChan <- event
	}()
}
