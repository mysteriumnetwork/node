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

package service

import (
	"encoding/json"
	"errors"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
)

var (
	// ErrorLocation error indicates that action (i.e. disconnect)
	ErrorLocation = errors.New("failed to detect service location")
	// ErrUnsupportedServiceType indicates that manager tried to create an unsupported service type
	ErrUnsupportedServiceType = errors.New("unsupported service type")
)

// Service interface represents pluggable Mysterium service
type Service interface {
	Serve(providerID identity.Identity) error
	Stop() error
	ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error)
}

// DialogWaiterFactory initiates communication channel which waits for incoming dialogs
type DialogWaiterFactory func(providerID identity.Identity, serviceType string) (communication.DialogWaiter, error)

// DialogHandlerFactory initiates instance which is able to handle incoming dialogs
type DialogHandlerFactory func(market.ServiceProposal, session.ConfigNegotiator) communication.DialogHandler

// DiscoveryFactory initiates instance which is able announce service discoverability
type DiscoveryFactory func() Discovery

// Discovery registers the service to the discovery api periodically
type Discovery interface {
	Start(ownIdentity identity.Identity, proposal market.ServiceProposal)
	Stop()
	Wait()
}

// NewManager creates new instance of pluggable instances manager
func NewManager(
	serviceRegistry *Registry,
	dialogWaiterFactory DialogWaiterFactory,
	dialogHandlerFactory DialogHandlerFactory,
	discoveryFactory DiscoveryFactory,
) *Manager {
	return &Manager{
		serviceRegistry:      serviceRegistry,
		servicePool:          NewPool(),
		dialogWaiterFactory:  dialogWaiterFactory,
		dialogHandlerFactory: dialogHandlerFactory,
		discoveryFactory:     discoveryFactory,
	}
}

// Manager entrypoint which knows how to start pluggable Mysterium instances
type Manager struct {
	dialogWaiterFactory  DialogWaiterFactory
	dialogHandlerFactory DialogHandlerFactory

	serviceRegistry *Registry
	servicePool     *Pool

	discoveryFactory DiscoveryFactory
}

// Start starts an instance of the given service type if knows one in service registry.
// It passes the options to the start method of the service.
// If an error occurs in the underlying service, the error is then returned.
func (manager *Manager) Start(providerID identity.Identity, serviceType string, options Options) (id ID, err error) {
	service, proposal, err := manager.serviceRegistry.Create(serviceType, options)
	if err != nil {
		return id, err
	}

	dialogWaiter, err := manager.dialogWaiterFactory(providerID, serviceType)
	if err != nil {
		return id, err
	}
	providerContact, err := dialogWaiter.Start()
	if err != nil {
		return id, err
	}
	proposal.SetProviderContact(providerID, providerContact)

	dialogHandler := manager.dialogHandlerFactory(proposal, service)
	if err = dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return id, err
	}

	discovery := manager.discoveryFactory()
	discovery.Start(providerID, proposal)

	instance := Instance{
		state:        Starting,
		options:      options,
		service:      service,
		proposal:     proposal,
		dialogWaiter: dialogWaiter,
		discovery:    discovery,
	}

	id, err = manager.servicePool.Add(&instance)
	if err != nil {
		return id, err
	}

	go func() {
		instance.state = Running
		serveErr := service.Serve(providerID)
		if serveErr != nil {
			log.Error("Service serve failed: ", serveErr)
		}

		instance.state = NotRunning

		stopErr := manager.servicePool.Stop(id)
		if stopErr != nil {
			log.Error("Service stop failed: ", stopErr)
		}

		discovery.Wait()
	}()

	return id, nil
}

// List returns array of running service instances.
func (manager *Manager) List() map[ID]*Instance {
	return manager.servicePool.List()
}

// Kill stops all services.
func (manager *Manager) Kill() error {
	return manager.servicePool.StopAll()
}

// Stop stops the service.
func (manager *Manager) Stop(id ID) error {
	return manager.servicePool.Stop(id)
}

// Service returns a service instance by requested id.
func (manager *Manager) Service(id ID) *Instance {
	return manager.servicePool.Instance(id)
}
