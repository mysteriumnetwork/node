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
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/market"
	discovery_registry "github.com/mysteriumnetwork/node/market/proposals/registry"
	"github.com/mysteriumnetwork/node/utils"
)

// Pool is responsible for supervising running instances
type Pool struct {
	instances []*Instance
}

// Instance represents a run service
type Instance struct {
	service      RunnableService
	proposal     market.ServiceProposal
	dialogWaiter communication.DialogWaiter
	discovery    *discovery_registry.Discovery
}

// RunnableService represents a runnable service
type RunnableService interface {
	Stop() error
}

// NewPool returns a empty service pool
func NewPool() *Pool {
	return &Pool{}
}

// Add registers a service to running instances pool
func (sr *Pool) Add(instance *Instance) {
	sr.instances = append(sr.instances, instance)
}

// Stop kills all sub-resources of instance
func (sr *Pool) Stop(instance *Instance) error {
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

	return errStop.Errorf("ErrorCollection(%s)", ", ")
}

// StopAll kills all running instances
func (sr *Pool) StopAll() error {
	errStop := utils.ErrorCollection{}
	for _, instance := range sr.instances {
		errStop.Add(sr.Stop(instance))
	}

	return errStop.Errorf("Some instances did not stop: %v", ". ")
}
