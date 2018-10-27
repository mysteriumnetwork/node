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
	"errors"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-manager] "

var (
	// ErrorLocation error indicates that action (i.e. disconnect)
	ErrorLocation = errors.New("failed to detect service location")
	// ErrUnsupportedServiceType indicates that manager tried to create an unsupported service type
	ErrUnsupportedServiceType = errors.New("unsupported service type")
)

// ServiceFactory initiates instance which is able to serve connections
type ServiceFactory func(Options) (Service, error)

// Service interface represents pluggable Mysterium service
type Service interface {
	Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.ConfigProvider, error)
	Wait() error
	Stop() error
}

// DialogWaiterFactory initiates communication channel which waits for incoming dialogs
type DialogWaiterFactory func(providerID identity.Identity) communication.DialogWaiter

// DialogHandlerFactory initiates instance which is able to handle incoming dialogs
type DialogHandlerFactory func(dto_discovery.ServiceProposal, session.ConfigProvider) communication.DialogHandler

// NewManager creates new instance of pluggable services manager
func NewManager(
	identityLoader identity_selector.Handler,
	serviceFactory ServiceFactory,
	dialogWaiterFactory DialogWaiterFactory,
	dialogHandlerFactory DialogHandlerFactory,
	discoveryService *discovery.Discovery,
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

	dialogWaiterFactory  func(identity identity.Identity) communication.DialogWaiter
	dialogWaiter         communication.DialogWaiter
	dialogHandlerFactory DialogHandlerFactory

	serviceFactory ServiceFactory
	service        Service

	discovery *discovery.Discovery
}

// Start starts service - does not block
func (manager *Manager) Start(options Options) (err error) {
	loadIdentity := identity_selector.NewLoader(manager.identityHandler, options.Identity, options.Passphrase)
	providerID, err := loadIdentity()
	if err != nil {
		return err
	}

	manager.service, err = manager.serviceFactory(options)
	if err != nil {
		return err
	}
	proposal, sessionConfigProvider, err := manager.service.Start(providerID)
	if err != nil {
		return err
	}

	manager.dialogWaiter = manager.dialogWaiterFactory(providerID)
	providerContact, err := manager.dialogWaiter.Start()
	if err != nil {
		return err
	}
	proposal.SetProviderContact(providerID, providerContact)

	dialogHandler := manager.dialogHandlerFactory(proposal, sessionConfigProvider)
	if err = manager.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return err
	}

	manager.discovery.Start(providerID, proposal)
	return nil
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	log.Info(logPrefix, "Waiting for discovery service to finish")
	manager.discovery.Wait()

	log.Info(logPrefix, "Waiting for service to finish")
	return manager.service.Wait()
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
	errService = manager.service.Stop()

	if errDialogWaiter != nil {
		return errDialogWaiter
	}
	if errService != nil {
		return errService
	}
	return nil
}
