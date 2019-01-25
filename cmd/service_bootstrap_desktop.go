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
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/proposals/registry"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/mapping"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_discovery "github.com/mysteriumnetwork/node/services/openvpn/discovery"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service bootstrap] "

// bootstrapServices loads all the components required for running services
func (di *Dependencies) bootstrapServices(nodeOptions node.Options) {
	di.bootstrapServiceComponents(nodeOptions)

	di.bootstrapServiceOpenvpn(nodeOptions)
	di.bootstrapServiceNoop(nodeOptions)
	di.bootstrapServiceWireguard(nodeOptions)
}

func (di *Dependencies) bootstrapServiceWireguard(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		wireguard.ServiceType,
		func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
			location, err := di.resolveIPsAndLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			wgOptions := serviceOptions.(wireguard_service.Options)

			mapPort := func(port int) func() {
				return mapping.GetPortMappingFunc(
					location.PubIP,
					location.OutIP,
					"UDP",
					port,
					"Myst node wireguard(tm) port mapping",
					di.EventBus)
			}

			return wireguard_service.NewManager(location, di.NATService, mapPort, wgOptions),
				wireguard_service.GetProposal(location.Country), nil
		},
	)
}

func (di *Dependencies) resolveIPsAndLocation() (loc location.ServiceLocationInfo, err error) {
	pubIP, err := di.IPResolver.GetPublicIP()
	if err != nil {
		return
	}
	loc.PubIP = pubIP

	outboundIP, err := di.IPResolver.GetOutboundIP()
	if err != nil {
		return
	}
	loc.OutIP = outboundIP

	currentCountry, err := di.LocationResolver.ResolveCountry(pubIP)
	if err != nil {
		log.Warn(logPrefix, "Failed to detect service country. ", err)
		err = service.ErrorLocation
		return
	}
	loc.Country = currentCountry

	log.Info(logPrefix, "Detected service country: ", loc.Country)
	return
}

func (di *Dependencies) bootstrapServiceOpenvpn(nodeOptions node.Options) {
	createService := func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
		location, err := di.resolveIPsAndLocation()
		if err != nil {
			return nil, market.ServiceProposal{}, err
		}

		currentLocation := market.Location{Country: location.Country}
		transportOptions := serviceOptions.(openvpn_service.Options)

		mapPort := func() func() {
			return mapping.GetPortMappingFunc(
				location.PubIP,
				location.OutIP,
				transportOptions.OpenvpnProtocol,
				transportOptions.OpenvpnPort,
				"Myst node OpenVPN port mapping",
				di.EventBus)
		}

		proposal := openvpn_discovery.NewServiceProposalWithLocation(currentLocation, transportOptions.OpenvpnProtocol)
		natService := nat.NewService()
		return openvpn_service.NewManager(nodeOptions, transportOptions, location, di.ServiceSessionStorage, natService, di.NATPinger, mapPort, di.LastSessionShutdown, di.NATTracker), proposal, nil
	}
	di.ServiceRegistry.Register(service_openvpn.ServiceType, createService)
}

func (di *Dependencies) bootstrapServiceNoop(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		service_noop.ServiceType,
		func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
			location, err := di.resolveIPsAndLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			return service_noop.NewManager(), service_noop.GetProposal(location.Country), nil
		},
	)
}

// bootstrapServiceComponents initiates ServicesManager dependency
func (di *Dependencies) bootstrapServiceComponents(nodeOptions node.Options) {
	di.NATService = nat.NewService()
	if err := di.NATService.Enable(); err != nil {
		log.Warn(logPrefix, "Failed to enable NAT forwarding: ", err)
	}
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
	newDialogHandler := func(proposal market.ServiceProposal, configProvider session.ConfigNegotiator) communication.DialogHandler {
		sessionManagerFactory := newSessionManagerFactory(
			proposal, di.ServiceSessionStorage,
			di.PromiseStorage,
			nodeOptions,
			di.NATPinger.PingTargetChan,
			di.LastSessionShutdown,
			di.NATTracker)
		return session.NewDialogHandler(sessionManagerFactory, configProvider.ProvideConfig, di.PromiseStorage)
	}
	newDiscovery := func() *registry.Discovery {
		return registry.NewService(di.IdentityRegistry, di.IdentityRegistration, di.MysteriumAPI, di.SignerFactory)
	}
	di.ServicesManager = service.NewManager(
		di.ServiceRegistry,
		newDialogWaiter,
		newDialogHandler,
		newDiscovery,
		di.NATPinger,
	)
}
