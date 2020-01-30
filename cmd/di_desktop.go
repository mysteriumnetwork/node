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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mmn"
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
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/ui"
	uinoop "github.com/mysteriumnetwork/node/ui/noop"
	"github.com/rs/zerolog/log"

	"github.com/pkg/errors"
)

// bootstrapServices loads all the components required for running services
func (di *Dependencies) bootstrapServices(nodeOptions node.Options, servicesOptions config.ServicesOptions) error {
	err := di.bootstrapServiceComponents(nodeOptions, servicesOptions)
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
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}
			outIP, err := di.IPResolver.GetOutboundIPAsString()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			wgOptions := serviceOptions.(wireguard_service.Options)

			var portPool port.ServicePortSupplier
			if wgOptions.Ports.IsSpecified() {
				log.Info().Msgf("Fixed service port range (%s) configured, using custom port pool", wgOptions.Ports)
				portPool = port.NewFixedRangePool(*wgOptions.Ports)
			} else {
				portPool = port.NewPool()
			}

			locationInfo := location.ServiceLocationInfo{
				OutIP:   outIP,
				PubIP:   loc.IP,
				Country: loc.Country,
			}

			portmapConfig := mapping.DefaultConfig()
			portMapper := mapping.NewPortMapper(portmapConfig, di.EventBus)

			svc := wireguard_service.NewManager(
				di.IPResolver,
				locationInfo,
				di.NATService,
				di.NATPinger,
				di.NATTracker,
				di.EventBus,
				wgOptions,
				portPool,
				portMapper)
			return svc, wireguard_service.GetProposal(loc), nil
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

		portMapper := mapping.NewPortMapper(mapping.DefaultConfig(), di.EventBus)

		manager := openvpn_service.NewManager(
			nodeOptions,
			transportOptions,
			locationInfo,
			di.ServiceSessionStorage,
			di.NATService,
			di.NATPinger,
			di.NATTracker,
			portPool,
			di.EventBus,
			portMapper,
		)
		return manager, proposal, nil
	}
	di.ServiceRegistry.Register(service_openvpn.ServiceType, createService)
}

func (di *Dependencies) bootstrapServiceNoop(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		service_noop.ServiceType,
		func(serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			return service_noop.NewManager(), service_noop.GetProposal(loc), nil
		},
	)
}

func (di *Dependencies) bootstrapProviderRegistrar(nodeOptions node.Options) error {
	cfg := registry.ProviderRegistrarConfig{
		MaxRetries:          nodeOptions.Transactor.ProviderMaxRegistrationAttempts,
		Stake:               nodeOptions.Transactor.ProviderRegistrationStake,
		DelayBetweenRetries: nodeOptions.Transactor.ProviderRegistrationRetryDelay,
		AccountantAddress:   common.HexToAddress(nodeOptions.Accountant.AccountantID),
		RegistryAddress:     common.HexToAddress(nodeOptions.Transactor.RegistryAddress),
	}
	di.ProviderRegistrar = registry.NewProviderRegistrar(di.Transactor, di.IdentityRegistry, cfg)
	return di.ProviderRegistrar.Subscribe(di.EventBus)
}

func (di *Dependencies) bootstrapAccountantPromiseSettler(nodeOptions node.Options) error {
	cfg := pingpong.AccountantPromiseSettlerConfig{
		AccountantAddress:    common.HexToAddress(nodeOptions.Accountant.AccountantID),
		Threshold:            nodeOptions.Payments.AccountantPromiseSettlingThreshold,
		MaxWaitForSettlement: nodeOptions.Payments.SettlementTimeout,
	}
	settler := pingpong.NewAccountantPromiseSettler(di.Transactor, di.AccountantPromiseStorage, di.BCHelper, di.IdentityRegistry, di.Keystore, di.AccountantPromiseStorage, cfg)
	di.AccountantPromiseSettler = settler
	return settler.Subscribe(di.EventBus)
}

// bootstrapServiceComponents initiates ServicesManager dependency
func (di *Dependencies) bootstrapServiceComponents(nodeOptions node.Options, servicesOptions config.ServicesOptions) error {
	di.NATService = nat.NewService()
	if err := di.NATService.Enable(); err != nil {
		log.Warn().Err(err).Msg("Failed to enable NAT forwarding")
	}
	di.ServiceRegistry = service.NewRegistry()

	storage := session.NewEventBasedStorage(di.EventBus, session.NewStorageMemory())
	if err := storage.Subscribe(); err != nil {
		return errors.Wrap(err, "could not subscribe session to node events")
	}
	di.ServiceSessionStorage = storage

	di.PolicyRepository = policy.NewRepository(di.HTTPClient, servicesOptions.AccessPolicyAddress, servicesOptions.AccessPolicyFetchInterval)
	di.PolicyRepository.Start()

	newDialogWaiter := func(providerID identity.Identity, serviceType string, policies *[]market.AccessPolicy) (communication.DialogWaiter, error) {
		return nats_dialog.NewDialogWaiter(
			di.BrokerConnection,
			fmt.Sprintf("%v.%v", providerID.Address, serviceType),
			di.SignerFactory(providerID),
			policy.ValidateAllowedIdentity(di.PolicyRepository, policies),
		), nil
	}
	newDialogHandler := func(proposal market.ServiceProposal, configProvider session.ConfigProvider, serviceID string) (communication.DialogHandler, error) {
		sessionManagerFactory := newSessionManagerFactory(
			nodeOptions,
			proposal,
			di.ServiceSessionStorage,
			di.ProviderInvoiceStorage,
			di.AccountantPromiseStorage,
			di.PromiseStorage,
			di.NATPinger.PingTarget,
			di.NATTracker,
			serviceID,
			di.EventBus,
			di.BCHelper,
			di.Transactor,
			nodeOptions.Payments.PaymentsDisabled,
		)

		return session.NewDialogHandler(
			sessionManagerFactory,
			configProvider,
			di.PromiseStorage,
			identity.FromAddress(proposal.ProviderID),
			connectivity.NewStatusSubscriber(di.SessionConnectivityStatusStorage),
		), nil
	}

	di.ServicesManager = service.NewManager(
		di.ServiceRegistry,
		newDialogWaiter,
		newDialogHandler,
		di.DiscoveryFactory,
		di.EventBus,
		di.PolicyRepository,
	)

	serviceCleaner := service.Cleaner{SessionStorage: di.ServiceSessionStorage}
	if err := di.EventBus.Subscribe(service.StatusTopic, serviceCleaner.HandleServiceStatus); err != nil {
		log.Error().Msg("Failed to subscribe service cleaner")
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
	connFactory := func() (connection.Connection, error) {
		return wireguard_connection.NewConnection(nodeOptions.Directories.Config, di.IPResolver, di.NATPinger)
	}
	di.ConnectionRegistry.Register(wireguard.ServiceType, connFactory)
}

func (di *Dependencies) bootstrapUIServer(options node.Options) {
	if options.UI.UIEnabled {
		di.UIServer = ui.NewServer(options.BindAddress, options.UI.UIPort, options.TequilapiPort, di.JWTAuthenticator, di.HTTPClient)
		return
	}

	di.UIServer = uinoop.NewServer()
}

func (di *Dependencies) bootstrapMMN(options node.Options) {
	if !options.MMN.Enabled {
		return
	}

	collector := mmn.NewCollector(di.IPResolver)
	if err := collector.CollectEnvironmentInformation(); err != nil {
		log.Error().Msgf("Failed to collect environment information for MMN: %v", err)
		return
	}

	client := mmn.NewClient(di.HTTPClient, options.MMN.Address, di.SignerFactory)
	m := mmn.NewMMN(collector, client)

	if err := m.Subscribe(di.EventBus); err != nil {
		log.Error().Msgf("Failed to subscribe to events for MMN: %v", err)
	}
}
