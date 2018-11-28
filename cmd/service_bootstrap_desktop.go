// +build darwin windows linux

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

package cmd

import (
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/core/node"
	promise_noop "github.com/mysteriumnetwork/node/core/promise/methods/noop"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	service_wireguard "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/session"
)

func (di *Dependencies) bootstrapServices(nodeOptions node.Options) {
	di.bootstrapServiceComponents(nodeOptions)

	di.bootstrapServiceOpenvpn(nodeOptions)
	di.bootstrapServiceNoop(nodeOptions)
	di.bootstrapServiceWireguard(nodeOptions)
}

func (di *Dependencies) bootstrapServiceOpenvpn(nodeOptions node.Options) {
	createService := func(serviceOptions service.Options) (service.Service, error) {
		transportOptions := serviceOptions.Options.(openvpn_service.Options)
		return openvpn_service.NewManager(nodeOptions, transportOptions, di.IPResolver, di.LocationResolver, di.ServiceSessionStorage), nil
	}
	di.ServiceRegistry.Register(service_openvpn.ServiceType, createService)

	di.ServiceRunner.Register(service_openvpn.ServiceType)
}

func (di *Dependencies) bootstrapServiceNoop(nodeOptions node.Options) {
	di.ServiceRegistry.Register(service_noop.ServiceType, func(serviceOptions service.Options) (service.Service, error) {
		return service_noop.NewManager(di.LocationResolver, di.IPResolver), nil
	})

	di.ServiceRunner.Register(service_noop.ServiceType)
}

func (di *Dependencies) bootstrapServiceWireguard(nodeOptions node.Options) {
	di.ServiceRegistry.Register(service_wireguard.ServiceType, func(serviceOptions service.Options) (service.Service, error) {
		return service_wireguard.NewManager(di.LocationResolver, di.IPResolver), nil
	})

	di.ServiceRunner.Register(service_wireguard.ServiceType)
}

// bootstrapServiceComponents initiates ServiceManager dependency
func (di *Dependencies) bootstrapServiceComponents(nodeOptions node.Options) {
	identityHandler := identity_selector.NewHandler(
		di.IdentityManager,
		di.MysteriumClient,
		identity.NewIdentityCache(nodeOptions.Directories.Keystore, "remember.json"),
		di.SignerFactory,
	)

	di.ServiceRegistry = service.NewRegistry()
	di.ServiceSessionStorage = session.NewStorageMemory()

	newDialogWaiter := func(providerID identity.Identity, serviceType string) (communication.DialogWaiter, error) {
		address, err := nats_discovery.NewAddressFromHostAndID(di.NetworkDefinition.BrokerAddress, providerID, serviceType)
		if err != nil {
			return nil, err
		}

		return nats_dialog.NewDialogWaiter(
			address,
			di.SignerFactory(providerID),
			di.IdentityRegistry,
		), nil
	}
	newDialogHandler := func(proposal dto_discovery.ServiceProposal, configProvider session.ConfigProvider) communication.DialogHandler {
		promiseHandler := func(dialog communication.Dialog) session.PromiseProcessor {
			if nodeOptions.ExperimentPromiseCheck {
				return &promise_noop.FakePromiseEngine{}
			}
			return promise_noop.NewPromiseProcessor(dialog, identity.NewBalance(di.EtherClient), di.Storage)
		}
		sessionManagerFactory := newSessionManagerFactory(proposal, configProvider, di.ServiceSessionStorage, promiseHandler)
		return session.NewDialogHandler(sessionManagerFactory)
	}

	runnableServiceFactory := func() service.RunnableService {
		return service.NewManager(
			identityHandler,
			di.ServiceRegistry.Create,
			newDialogWaiter,
			newDialogHandler,
			discovery.NewService(di.IdentityRegistry, di.IdentityRegistration, di.MysteriumClient, di.SignerFactory),
		)
	}

	di.ServiceRunner = service.NewRunner(runnableServiceFactory)
}
