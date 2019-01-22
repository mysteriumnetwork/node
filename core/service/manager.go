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

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/proposals/registry"
	"github.com/mysteriumnetwork/node/session"
)

var (
	// ErrorLocation error indicates that action (i.e. disconnect)
	ErrorLocation = errors.New("failed to detect service location")
	// ErrUnsupportedServiceType indicates that manager tried to create an unsupported service type
	ErrUnsupportedServiceType = errors.New("unsupported service type")
)

// ServiceFactory initiates instance which is able to serve connections
type ServiceFactory func(Options) (Service, market.ServiceProposal, error)

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

// NewManager creates new instance of pluggable services manager
func NewManager(
	identityLoader identity_selector.Handler,
	serviceFactory ServiceFactory,
	dialogWaiterFactory DialogWaiterFactory,
	dialogHandlerFactory DialogHandlerFactory,
	discoveryService *registry.Discovery,
) *Manager {
	return &Manager{
		identityHandler:      identityLoader,
		serviceFactory:       serviceFactory,
		dialogWaiterFactory:  dialogWaiterFactory,
		dialogHandlerFactory: dialogHandlerFactory,
		discovery:            discoveryService,
	}
}

// Manager entrypoint which knows how to start pluggable Mysterium services
type Manager struct {
	identityHandler identity_selector.Handler

	dialogWaiterFactory  DialogWaiterFactory
	dialogWaiter         communication.DialogWaiter
	dialogHandlerFactory DialogHandlerFactory

	serviceFactory ServiceFactory
	service        Service

	discovery *registry.Discovery
}

// Start starts a service of the given service type if it has one. The method blocks.
// It passes the options to the start method of the service.
// If an error occurs in the underlying service, the error is then returned.
func (manager *Manager) Start(options Options) (err error) {
	loadIdentity := identity_selector.NewLoader(manager.identityHandler, options.Identity, options.Passphrase)
	providerID, err := loadIdentity()
	if err != nil {
		return err
	}

	service, proposal, err := manager.serviceFactory(options)
	if err != nil {
		return err
	}
	manager.service = service

	manager.dialogWaiter, err = manager.dialogWaiterFactory(providerID, proposal.ServiceType)
	if err != nil {
		return err
	}
	providerContact, err := manager.dialogWaiter.Start()
	if err != nil {
		return err
	}
	proposal.SetProviderContact(providerID, providerContact)

	dialogHandler := manager.dialogHandlerFactory(proposal, service)
	if err = manager.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return err
	}

	manager.discovery.Start(providerID, proposal)

	err = manager.service.Serve(providerID)
	manager.discovery.Wait()
	return err
}

// Kill stops service
func (manager *Manager) Kill() error {
	var errDialogWaiter, errService error

	if manager.discovery != nil {
		manager.discovery.Stop()
	}
	if manager.dialogWaiter != nil {
		errDialogWaiter = manager.dialogWaiter.Stop()
	}
	if manager.service != nil {
		errService = manager.service.Stop()
	}

	if errDialogWaiter != nil {
		return errDialogWaiter
	}
	if errService != nil {
		return errService
	}
	return nil
}
