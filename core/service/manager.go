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
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/metadata"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-manager] "

var (
	// ErrorLocation error indicates that action (i.e. disconnect)
	ErrorLocation = errors.New("failed to detect service location")
)

// Service interface represents pluggable Mysterium service
type Service interface {
	Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.Manager, error)
	Wait() error
	Stop() error
}

// NewManager creates new instance of pluggable services manager
func NewManager(
	networkDefinition metadata.NetworkDefinition,
	identityLoader identity_selector.Loader,
	signerFactory identity.SignerFactory,
	identityRegistry identity_registry.IdentityRegistry,
	service Service,
	discoveryService *discovery.Discovery,
) *Manager {
	return &Manager{
		identityLoader: identityLoader,
		dialogWaiterFactory: func(providerID identity.Identity) communication.DialogWaiter {
			return nats_dialog.NewDialogWaiter(
				nats_discovery.NewAddressGenerate(networkDefinition.BrokerAddress, providerID),
				signerFactory(providerID),
				identityRegistry,
			)
		},
		service:   service,
		discovery: discoveryService,
	}
}

// Manager entrypoint which knows how to start pluggable Mysterium services
type Manager struct {
	identityLoader identity_selector.Loader

	dialogWaiterFactory func(identity identity.Identity) communication.DialogWaiter
	dialogWaiter        communication.DialogWaiter

	service   Service
	discovery *discovery.Discovery
}

// Start starts service - does not block
func (manager *Manager) Start() (err error) {
	providerID, err := manager.identityLoader()
	if err != nil {
		return err
	}

	proposal, sessionManager, err := manager.service.Start(providerID)
	if err != nil {
		return err
	}

	manager.dialogWaiter = manager.dialogWaiterFactory(providerID)
	providerContact, err := manager.dialogWaiter.Start()
	if err != nil {
		return err
	}
	proposal.SetProviderContact(providerID, providerContact)

	dialogHandler := session.NewDialogHandler(proposal.ID, sessionManager)
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
