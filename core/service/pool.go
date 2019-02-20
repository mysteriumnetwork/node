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
	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/market"
	discovery_registry "github.com/mysteriumnetwork/node/market/proposals/registry"
	"github.com/mysteriumnetwork/node/utils"
)

// ID represent unique identifier of the running service.
type ID string

// RunnableService represents a runnable service
type RunnableService interface {
	Stop() error
}

// Pool is responsible for supervising running instances
type Pool struct {
	instances map[ID]*Instance
}

// NewPool returns a empty service pool
func NewPool() *Pool {
	return &Pool{
		instances: make(map[ID]*Instance),
	}
}

// Add registers a service to running instances pool
func (p *Pool) Add(instance *Instance) (ID, error) {
	id, err := generateID()
	if err != nil {
		return id, err
	}

	p.instances[id] = instance
	return id, nil
}

// Del removes a service from running instances pool
func (p *Pool) Del(id ID) {
	delete(p.instances, id)
}

// Stop kills all sub-resources of instance
func (p *Pool) Stop(id ID) error {
	instance := p.instances[id]
	errStop := utils.ErrorCollection{}
	if instance.discovery != nil {
		instance.discovery.Stop()
	}
	if instance.dialogWaiter != nil {
		errStop.Add(instance.dialogWaiter.Stop())
	}
	if instance.service != nil {
		errStop.Add(instance.service.Stop())
	}

	p.Del(id)
	return errStop.Errorf("ErrorCollection(%s)", ", ")
}

// StopAll kills all running instances
func (p *Pool) StopAll() error {
	errStop := utils.ErrorCollection{}
	for id := range p.instances {
		errStop.Add(p.Stop(id))
	}

	return errStop.Errorf("Some instances did not stop: %v", ". ")
}

// List returns all running service instances.
func (p *Pool) List() map[ID]*Instance {
	return p.instances
}

// Instance returns service instance by the requested id.
func (p *Pool) Instance(id ID) *Instance {
	return p.instances[id]
}

// Instance represents a run service
type Instance struct {
	state        State
	service      RunnableService
	proposal     market.ServiceProposal
	dialogWaiter communication.DialogWaiter
	discovery    *discovery_registry.Discovery
}

// Proposal returns service proposal of the running service instance.
func (i *Instance) Proposal() market.ServiceProposal {
	return i.proposal
}

// State returns the service instance state.
func (i *Instance) State() State {
	return i.state
}

func generateID() (ID, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return ID(""), err
	}
	return ID(uid.String()), nil
}
