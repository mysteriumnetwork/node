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

package service

import (
	"sync"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/utils"
)

// ID represent unique identifier of the running service.
type ID string

// Pool is responsible for supervising running instances
type Pool struct {
	eventPublisher Publisher
	instances      map[ID]*Instance
	sync.Mutex
}

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, data interface{})
}

// NewPool returns a empty service pool
func NewPool(eventPublisher Publisher) *Pool {
	return &Pool{
		eventPublisher: eventPublisher,
		instances:      make(map[ID]*Instance),
	}
}

// Add registers a service to running instances pool
func (p *Pool) Add(instance *Instance) {
	p.Lock()
	defer p.Unlock()

	p.instances[instance.ID] = instance
}

// Del removes a service from running instances pool
func (p *Pool) Del(id ID) {
	p.Lock()
	defer p.Unlock()
	p.del(id)
}

func (p *Pool) del(id ID) {
	delete(p.instances, id)
}

// ErrNoSuchInstance represents the error when we're stopping an instance that does not exist
var ErrNoSuchInstance = errors.New("no such instance")

// Stop kills all sub-resources of instance
func (p *Pool) Stop(id ID) error {
	p.Lock()
	defer p.Unlock()
	return p.stop(id)
}

func (p *Pool) stop(id ID) error {
	instance, ok := p.instances[id]
	if !ok {
		return ErrNoSuchInstance
	}
	p.del(id)
	return instance.stop()
}

// StopAll kills all running instances
func (p *Pool) StopAll() error {
	p.Lock()
	defer p.Unlock()
	errStop := utils.ErrorCollection{}
	for id := range p.instances {
		errStop.Add(p.stop(id))
	}

	return errStop.Errorf("Some instances did not stop: %v", ". ")
}

// List returns all running service instances.
func (p *Pool) List() []*Instance {
	p.Lock()
	defer p.Unlock()

	list := make([]*Instance, 0, len(p.instances))
	for _, instance := range p.instances {
		list = append(list, instance)
	}

	return list
}

// Instance returns service instance by the requested id.
func (p *Pool) Instance(id ID) *Instance {
	p.Lock()
	defer p.Unlock()
	return p.instances[id]
}

// NewInstance creates new instance of the service.
func NewInstance(
	providerID identity.Identity,
	serviceType string,
	options Options,
	proposal market.ServiceProposal,
	state servicestate.State,
	service Service,
	policyProvider policy.Provider,
	discovery Discovery,
) *Instance {
	return &Instance{
		ProviderID:     providerID,
		Type:           serviceType,
		Options:        options,
		Proposal:       proposal,
		state:          state,
		service:        service,
		policyProvider: policyProvider,
		discovery:      discovery,
	}
}

// Instance represents a run service
type Instance struct {
	ID         ID
	state      servicestate.State
	stateLock  sync.RWMutex
	ProviderID identity.Identity
	Type       string
	Options    Options
	service    Service

	muProposal      sync.Mutex
	Proposal        market.ServiceProposal
	policyProvider  policy.Provider
	discovery       Discovery
	eventPublisher  Publisher
	p2pChannelsLock sync.Mutex
	p2pChannels     []p2p.Channel
	location        locationResolver
}

// Service returns the running service implementation.
func (i *Instance) Service() Service {
	return i.service
}

// PolicyProvider returns policy provider implementation.
func (i *Instance) PolicyProvider() policy.Provider {
	return i.policyProvider
}

// State returns the service instance state.
func (i *Instance) State() servicestate.State {
	i.stateLock.RLock()
	defer i.stateLock.RUnlock()
	return i.state
}

func (i *Instance) proposalWithCurrentLocation() market.ServiceProposal {
	location, err := i.location.DetectLocation()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get current location for proposal, using last known location")
		return i.Proposal
	}

	i.muProposal.Lock()
	i.Proposal.Location = *market.NewLocation(location)
	i.muProposal.Unlock()

	return i.Proposal
}

func (i *Instance) setState(newState servicestate.State) {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	i.state = newState

	i.eventPublisher.Publish(servicestate.AppTopicServiceStatus, i.toEvent())
}

func (i *Instance) addP2PChannel(ch p2p.Channel) {
	i.p2pChannelsLock.Lock()
	defer i.p2pChannelsLock.Unlock()

	i.p2pChannels = append(i.p2pChannels, ch)
}

func (i *Instance) stop() error {
	errStop := utils.ErrorCollection{}
	if i.discovery != nil {
		i.discovery.Stop()
	}
	if i.service != nil {
		errStop.Add(i.service.Stop())
	}

	i.p2pChannelsLock.Lock()
	for _, channel := range i.p2pChannels {
		errStop.Add(channel.Close())
	}
	i.p2pChannelsLock.Unlock()

	i.setState(servicestate.NotRunning)
	return errStop.Errorf("ErrorCollection(%s)", ", ")
}

// toEvent returns an event representation of the instance
func (i *Instance) toEvent() servicestate.AppEventServiceStatus {
	return servicestate.AppEventServiceStatus{
		ID:         string(i.ID),
		ProviderID: i.Proposal.ProviderID,
		Type:       i.Proposal.ServiceType,
		Status:     string(i.state),
	}
}

// CopyProposal returns a copy of Proposal
func (i *Instance) CopyProposal() market.ServiceProposal {
	i.muProposal.Lock()
	defer i.muProposal.Unlock()

	var proposal market.ServiceProposal
	if err := copier.CopyWithOption(&proposal, i.Proposal, copier.Option{DeepCopy: true}); err != nil {
		panic(err)
	}
	// workaround b/c of copier bug: it make empty slice instead of nil
	if i.Proposal.Contacts == nil {
		proposal.Contacts = nil
	}

	return proposal
}
