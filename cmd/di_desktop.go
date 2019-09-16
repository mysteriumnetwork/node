// +build darwin windows linux,!android

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
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/mapping"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_discovery "github.com/mysteriumnetwork/node/services/openvpn/discovery"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/ui"
	uinoop "github.com/mysteriumnetwork/node/ui/noop"
	"github.com/pkg/errors"
)

// bootstrapServices loads all the components required for running services
func (di *Dependencies) bootstrapServices(nodeOptions node.Options) error {
	err := di.bootstrapServiceComponents(nodeOptions)
	if err != nil {
		return errors.Wrap(err, "service bootstrap failed")
	}

	di.bootstrapServiceOpenvpn(nodeOptions)
	di.bootstrapServiceNoop(nodeOptions)
	di.bootstrapServiceWireguard(nodeOptions)

	return nil
}

func (di *Dependencies) bootstrapServiceWireguard(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		wireguard.ServiceType,
		func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
			location, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}
			outIP, err := di.IPResolver.GetOutboundIPAsString()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			wgOptions := serviceOptions.(wireguard_service.Options)

			mapPort := func(port int) func() {
				return mapping.GetPortMappingFunc(
					location.IP,
					outIP,
					"UDP",
					port,
					"Myst node wireguard(tm) port mapping",
					di.EventBus)
			}

			var portPool port.ServicePortSupplier
			if wgOptions.Ports.IsSpecified() {
				log.Infof("fixed service port range (%s) configured, using custom port pool", wgOptions.Ports)
				portPool = port.NewFixedRangePool(*wgOptions.Ports)
			} else {
				portPool = port.NewPool()
			}

			return wireguard_service.NewManager(di.IPResolver, di.NATService, mapPort, wgOptions, portPool),
				wireguard_service.GetProposal(location), nil
		},
	)
}

func (di *Dependencies) bootstrapServiceOpenvpn(nodeOptions node.Options) {
	createService := func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
		if err := nodeOptions.Openvpn.Check(); err != nil {
			return nil, market.ServiceProposal{}, err
		}

		loc, err := di.LocationResolver.DetectLocation()
		if err != nil {
			return nil, market.ServiceProposal{}, err
		}
		outIP, err := di.IPResolver.GetOutboundIPAsString()
		if err != nil {
			return nil, market.ServiceProposal{}, err
		}

		currentLocation := market.Location{
			Continent: loc.Continent,
			Country:   loc.Country,
			City:      loc.City,

			ASN:      loc.ASN,
			ISP:      loc.ISP,
			NodeType: loc.NodeType,
		}

		transportOptions := serviceOptions.(openvpn_service.Options)

		mapPort := func(port int) func() {
			return mapping.GetPortMappingFunc(
				loc.IP,
				outIP,
				transportOptions.Protocol,
				port,
				"Myst node OpenVPN port mapping",
				di.EventBus)
		}

		locationInfo := location.ServiceLocationInfo{
			OutIP:   outIP,
			PubIP:   loc.IP,
			Country: loc.Country,
		}

		proposal := openvpn_discovery.NewServiceProposalWithLocation(currentLocation, transportOptions.Protocol)

		var portPool port.ServicePortSupplier
		if transportOptions.Port != 0 {
			portPool = port.NewPoolFixed(port.Port(transportOptions.Port))
		} else {
			portPool = port.NewPool()
		}

		manager := openvpn_service.NewManager(
			nodeOptions,
			transportOptions,
			locationInfo,
			di.ServiceSessionStorage,
			di.NATService,
			di.NATPinger,
			mapPort,
			di.NATTracker,
			portPool,
			di.EventBus,
		)
		return manager, proposal, nil
	}
	di.ServiceRegistry.Register(service_openvpn.ServiceType, createService)
}

func (di *Dependencies) bootstrapServiceNoop(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		service_noop.ServiceType,
		func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
			location, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			return service_noop.NewManager(), service_noop.GetProposal(location), nil
		},
	)
}

// bootstrapServiceComponents initiates ServicesManager dependency
func (di *Dependencies) bootstrapServiceComponents(nodeOptions node.Options) error {
	di.NATService = nat.NewService()
	if err := di.NATService.Enable(); err != nil {
		log.Warn("Failed to enable NAT forwarding: ", err)
	}
	di.ServiceRegistry = service.NewRegistry()
	storage := session.NewEventBasedStorage(di.EventBus, session.NewStorageMemory())
	di.ServiceSessionStorage = storage

	err := storage.Subscribe()
	if err != nil {
		return errors.Wrap(err, "could not bootstrap service components")
	}

	registeredIdentityValidator := func(peerID identity.Identity) error {
		registered, err := di.IdentityRegistry.IsRegistered(peerID)
		if err != nil {
			return err
		} else if !registered {
			return errors.New("identity is not registered")
		}
		return nil
	}

	newDialogWaiter := func(providerID identity.Identity, serviceType string, allowedIDs []identity.Identity) (communication.DialogWaiter, error) {
		address, err := nats_discovery.NewAddressFromHostAndID(di.NetworkDefinition.BrokerAddress, providerID, serviceType)
		if err != nil {
			return nil, err
		}

		allowedIdentityValidator := func(peerID identity.Identity) error {
			if len(allowedIDs) == 0 {
				return nil
			}

			for _, id := range allowedIDs {
				if peerID.Address == id.Address {
					return nil
				}
			}
			return errors.New("identity is not allowed")
		}

		return nats_dialog.NewDialogWaiter(
			address,
			di.SignerFactory(providerID),
			registeredIdentityValidator,
			allowedIdentityValidator,
		), nil
	}
	newDialogHandler := func(proposal market.ServiceProposal, configProvider session.ConfigNegotiator, serviceID string) (communication.DialogHandler, error) {
		sessionManagerFactory := newSessionManagerFactory(
			proposal,
			di.ServiceSessionStorage,
			di.PromiseStorage,
			di.NATPinger.PingTarget,
			di.NATTracker,
			serviceID,
			di.EventBus)
		return session.NewDialogHandler(sessionManagerFactory, configProvider.ProvideConfig, di.PromiseStorage, identity.FromAddress(proposal.ProviderID)), nil
	}
	di.ServicesManager = service.NewManager(
		di.ServiceRegistry,
		newDialogWaiter,
		newDialogHandler,
		di.DiscoveryFactory,
		di.EventBus,
		nodeOptions.BindAddress,
	)

	serviceCleaner := service.Cleaner{SessionStorage: di.ServiceSessionStorage}
	if err := di.EventBus.Subscribe(service.StatusTopic, serviceCleaner.HandleServiceStatus); err != nil {
		log.Error("failed to subscribe service cleaner")
	}

	return nil
}

func (di *Dependencies) registerConnections(nodeOptions node.Options) {
	di.registerOpenvpnConnection(nodeOptions)
	di.registerNoopConnection()
	di.registerWireguardConnection(nodeOptions)
}

func (di *Dependencies) registerWireguardConnection(nodeOptions node.Options) {
	wireguard.Bootstrap()
	di.ConnectionRegistry.Register(wireguard.ServiceType, wireguard_connection.NewConnectionCreator(nodeOptions.Directories.Config))
}

func (di *Dependencies) bootstrapUIServer(options node.Options) {
	if options.UI.UIEnabled {
		di.UIServer = ui.NewServer(options.BindAddress, options.UI.UIPort, options.TequilapiPort, di.JWTAuthenticator)
		return
	}

	di.UIServer = uinoop.NewServer()
}
