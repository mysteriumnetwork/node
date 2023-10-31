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
	"time"

	"github.com/mysteriumnetwork/node/core/policy/requested"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/policy/localcopy"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/mmn"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/scraping"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	netstack_provider "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack-provider"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/mysteriumnetwork/node/session/pingpong"
)

// bootstrapServices loads all the components required for running services
func (di *Dependencies) bootstrapServices(nodeOptions node.Options) error {
	if nodeOptions.Consumer {
		log.Debug().Msg("Skipping services bootstrap for consumer mode")
		return nil
	}

	err := di.bootstrapServiceComponents(nodeOptions)
	if err != nil {
		return errors.Wrap(err, "service bootstrap failed")
	}

	if config.GetBool(config.FlagUserspace) {
		netstack_provider.InitUserspaceShaper(di.EventBus)
	}
	di.bootstrapServiceOpenvpn(nodeOptions)
	di.bootstrapServiceNoop(nodeOptions)
	resourcesAllocator := resources.NewAllocator(di.PortPool, wireguard_service.GetOptions().Subnet)

	dnsHandler, err := dns.ResolveViaSystem()
	if err != nil {
		log.Error().Err(err).Msg("Provider DNS are not available")
		return err
	}

	di.dnsProxy = dns.NewProxy("", config.GetInt(config.FlagDNSListenPort), dnsHandler)

	// disable for mobile
	if !nodeOptions.Mobile {
		di.bootstrapServiceWireguard(nodeOptions, resourcesAllocator, di.WireguardClientFactory)
	}
	di.bootstrapServiceScraping(nodeOptions, resourcesAllocator, di.WireguardClientFactory)
	di.bootstrapServiceDataTransfer(nodeOptions, resourcesAllocator, di.WireguardClientFactory)
	di.bootstrapServiceDVPN(nodeOptions, resourcesAllocator, di.WireguardClientFactory)

	return nil
}

func (di *Dependencies) bootstrapServiceWireguard(nodeOptions node.Options, resourcesAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	di.ServiceRegistry.Register(
		wireguard.ServiceType,
		func(serviceOptions service.Options) (service.Service, error) {
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, err
			}

			svc := wireguard_service.NewManager(
				di.IPResolver,
				loc.Country,
				di.NATService,
				di.EventBus,
				di.ServiceFirewall,
				resourcesAllocator,
				wgClientFactory,
				di.dnsProxy,
			)
			return svc, nil
		},
	)
}

func (di *Dependencies) bootstrapServiceScraping(nodeOptions node.Options, resourcesAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	di.ServiceRegistry.Register(
		scraping.ServiceType,
		func(serviceOptions service.Options) (service.Service, error) {
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, err
			}

			svc := wireguard_service.NewManager(
				di.IPResolver,
				loc.Country,
				di.NATService,
				di.EventBus,
				di.ServiceFirewall,
				resourcesAllocator,
				wgClientFactory,
				di.dnsProxy,
			)
			return svc, nil
		},
	)
}

func (di *Dependencies) bootstrapServiceDataTransfer(nodeOptions node.Options, resourcesAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	di.ServiceRegistry.Register(
		datatransfer.ServiceType,
		func(serviceOptions service.Options) (service.Service, error) {
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, err
			}

			svc := wireguard_service.NewManager(
				di.IPResolver,
				loc.Country,
				di.NATService,
				di.EventBus,
				di.ServiceFirewall,
				resourcesAllocator,
				wgClientFactory,
				di.dnsProxy,
			)
			return svc, nil
		},
	)
}

func (di *Dependencies) bootstrapServiceDVPN(nodeOptions node.Options, resourcesAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	di.ServiceRegistry.Register(
		dvpn.ServiceType,
		func(serviceOptions service.Options) (service.Service, error) {
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, err
			}

			svc := wireguard_service.NewManager(
				di.IPResolver,
				loc.Country,
				di.NATService,
				di.EventBus,
				di.ServiceFirewall,
				resourcesAllocator,
				wgClientFactory,
				di.dnsProxy,
			)
			return svc, nil
		},
	)
}

func (di *Dependencies) bootstrapServiceOpenvpn(nodeOptions node.Options) {
	createService := func(serviceOptions service.Options) (service.Service, error) {
		if err := nodeOptions.Openvpn.Check(); err != nil {
			return nil, err
		}

		loc, err := di.LocationResolver.DetectLocation()
		if err != nil {
			return nil, err
		}

		transportOptions := serviceOptions.(openvpn_service.Options)

		manager := openvpn_service.NewManager(
			nodeOptions,
			transportOptions,
			loc.Country,
			di.IPResolver,
			di.ServiceSessions,
			di.NATService,
			di.PortPool,
			di.EventBus,
			di.ServiceFirewall,
		)
		return manager, nil
	}
	di.ServiceRegistry.Register(service_openvpn.ServiceType, createService)
}

func (di *Dependencies) bootstrapServiceNoop(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		service_noop.ServiceType,
		func(serviceOptions service.Options) (service.Service, error) {
			return service_noop.NewManager(), nil
		},
	)
}

func (di *Dependencies) bootstrapHermesPromiseSettler(nodeOptions node.Options) error {
	di.HermesChannelRepository = pingpong.NewHermesChannelRepository(
		di.HermesPromiseStorage,
		di.BCHelper,
		di.EventBus,
		di.BeneficiaryProvider,
		di.HermesCaller,
		di.AddressProvider,
		di.SignerFactory,
		di.Keystore,
	)

	if err := di.HermesChannelRepository.Subscribe(di.EventBus); err != nil {
		log.Error().Err(err).Msg("Failed to subscribe channel repository")
		return errors.Wrap(err, "could not subscribe channel repository to relevant events")
	}

	settler := pingpong.NewHermesPromiseSettler(
		di.Transactor,
		di.HermesPromiseStorage,
		di.HermesPromiseHandler,
		di.AddressProvider,
		func(hermesURL string) pingpong.HermesHTTPRequester {
			return pingpong.NewHermesCaller(di.HTTPClient, hermesURL)
		},
		di.HermesURLGetter,
		di.HermesChannelRepository,
		di.BCHelper,
		di.IdentityRegistry,
		di.Keystore,
		di.SettlementHistoryStorage,
		di.EventBus,
		di.ObserverAPI,
		di.BeneficiaryAddressStorage,
		pingpong.HermesPromiseSettlerConfig{
			BalanceThreshold:        nodeOptions.Payments.HermesPromiseSettlingThreshold,
			MaxFeeThreshold:         nodeOptions.Payments.MaxFeeSettlingThreshold,
			MinAutoSettleAmount:     nodeOptions.Payments.MinAutoSettleAmount,
			MaxUnSettledAmount:      nodeOptions.Payments.MaxUnSettledAmount,
			SettlementCheckTimeout:  nodeOptions.Payments.SettlementTimeout,
			SettlementCheckInterval: nodeOptions.Payments.SettlementRecheckInterval,
			L1ChainID:               nodeOptions.Chains.Chain1.ChainID,
			L2ChainID:               nodeOptions.Chains.Chain2.ChainID,
		},
	)
	if err := settler.Subscribe(di.EventBus); err != nil {
		return errors.Wrap(err, "could not subscribe promise settler to relevant events")
	}

	di.HermesPromiseSettler = settler
	return nil
}

// bootstrapServiceComponents initiates ServicesManager dependency
func (di *Dependencies) bootstrapServiceComponents(nodeOptions node.Options) error {
	di.NATService = nat.NewService()
	if err := di.NATService.Enable(); err != nil {
		log.Warn().Err(err).Msg("Failed to enable NAT forwarding")
	}
	di.ServiceRegistry = service.NewRegistry()

	di.ServiceSessions = service.NewSessionPool(di.EventBus)

	di.PolicyOracle = localcopy.NewOracle(
		di.HTTPClient,
		config.GetString(config.FlagAccessPolicyAddress),
		config.GetDuration(config.FlagAccessPolicyFetchInterval),
		config.GetBool(config.FlagAccessPolicyFetchingEnabled),
	)
	go di.PolicyOracle.Start()

	di.PolicyProvider = requested.NewRequestedProvider(
		di.HTTPClient,
		config.GetString(config.FlagAccessPolicyAddress),
	)

	di.HermesStatusChecker = pingpong.NewHermesStatusChecker(di.BCHelper, di.ObserverAPI, nodeOptions.Payments.HermesStatusRecheckInterval)

	newP2PSessionHandler := func(serviceInstance *service.Instance, channel p2p.Channel) *service.SessionManager {
		paymentEngineFactory := pingpong.InvoiceFactoryCreator(
			channel, nodeOptions.Payments.ProviderInvoiceFrequency, nodeOptions.Payments.ProviderLimitInvoiceFrequency,
			pingpong.PromiseWaitTimeout, di.ProviderInvoiceStorage,
			pingpong.DefaultHermesFailureCount,
			uint16(nodeOptions.Payments.MaxAllowedPaymentPercentile),
			nodeOptions.Payments.MaxUnpaidInvoiceValue,
			nodeOptions.Payments.LimitUnpaidInvoiceValue,
			di.HermesStatusChecker,
			di.EventBus,
			di.HermesPromiseHandler,
			di.AddressProvider,
			di.ObserverAPI,
		)
		return service.NewSessionManager(
			serviceInstance,
			di.ServiceSessions,
			paymentEngineFactory,
			di.EventBus,
			channel,
			service.DefaultConfig(),
			di.PricingHelper,
		)
	}

	di.ServicesManager = service.NewManager(
		di.ServiceRegistry,
		di.DiscoveryFactory,
		di.EventBus,
		di.PolicyOracle,
		di.PolicyProvider,
		di.P2PListener,
		newP2PSessionHandler,
		di.SessionConnectivityStatusStorage,
		di.LocationResolver,
	)

	serviceCleaner := service.Cleaner{SessionStorage: di.ServiceSessions}
	if err := di.EventBus.Subscribe(servicestate.AppTopicServiceStatus, serviceCleaner.HandleServiceStatus); err != nil {
		log.Error().Err(err).Msg("Failed to subscribe service cleaner")
	}

	return nil
}

func (di *Dependencies) registerConnections(nodeOptions node.Options) {
	di.registerOpenvpnConnection(nodeOptions)
	di.registerNoopConnection()

	resourceAllocator := resources.NewAllocator(nil, wireguard_service.DefaultOptions.Subnet)

	di.registerWireguardConnection(nodeOptions, resourceAllocator, di.WireguardClientFactory)
	di.registerScrapingConnection(nodeOptions, resourceAllocator, di.WireguardClientFactory)
	di.registerDataTransferConnection(nodeOptions, resourceAllocator, di.WireguardClientFactory)
	di.registerDVPNConnection(nodeOptions, resourceAllocator, di.WireguardClientFactory)
}

func (di *Dependencies) registerWireguardConnection(nodeOptions node.Options, resourceAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	wireguard.Bootstrap()
	handshakeWaiter := wireguard_connection.NewHandshakeWaiter()
	endpointFactory := func() (wireguard.ConnectionEndpoint, error) {
		return endpoint.NewConnectionEndpoint(resourceAllocator, wgClientFactory)
	}
	connFactory := func() (connection.Connection, error) {
		opts := wireguard_connection.Options{
			DNSScriptDir:     nodeOptions.Directories.Script,
			HandshakeTimeout: 1 * time.Minute,
		}
		return wireguard_connection.NewConnection(opts, di.IPResolver, endpointFactory, handshakeWaiter)
	}
	di.ConnectionRegistry.Register(wireguard.ServiceType, connFactory)
}

func (di *Dependencies) registerScrapingConnection(nodeOptions node.Options, resourceAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	scraping.Bootstrap()
	handshakeWaiter := wireguard_connection.NewHandshakeWaiter()
	endpointFactory := func() (wireguard.ConnectionEndpoint, error) {
		return endpoint.NewConnectionEndpoint(resourceAllocator, wgClientFactory)
	}
	connFactory := func() (connection.Connection, error) {
		opts := wireguard_connection.Options{
			DNSScriptDir:     nodeOptions.Directories.Script,
			HandshakeTimeout: 1 * time.Minute,
		}
		return wireguard_connection.NewConnection(opts, di.IPResolver, endpointFactory, handshakeWaiter)
	}
	di.ConnectionRegistry.Register(scraping.ServiceType, connFactory)
}

func (di *Dependencies) registerDataTransferConnection(nodeOptions node.Options, resourceAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	datatransfer.Bootstrap()
	handshakeWaiter := wireguard_connection.NewHandshakeWaiter()
	endpointFactory := func() (wireguard.ConnectionEndpoint, error) {
		return endpoint.NewConnectionEndpoint(resourceAllocator, wgClientFactory)
	}
	connFactory := func() (connection.Connection, error) {
		opts := wireguard_connection.Options{
			DNSScriptDir:     nodeOptions.Directories.Script,
			HandshakeTimeout: 1 * time.Minute,
		}
		return wireguard_connection.NewConnection(opts, di.IPResolver, endpointFactory, handshakeWaiter)
	}
	di.ConnectionRegistry.Register(datatransfer.ServiceType, connFactory)
}

func (di *Dependencies) registerDVPNConnection(nodeOptions node.Options, resourceAllocator *resources.Allocator, wgClientFactory *endpoint.WgClientFactory) {
	dvpn.Bootstrap()
	handshakeWaiter := wireguard_connection.NewHandshakeWaiter()
	endpointFactory := func() (wireguard.ConnectionEndpoint, error) {
		return endpoint.NewConnectionEndpoint(resourceAllocator, wgClientFactory)
	}
	connFactory := func() (connection.Connection, error) {
		opts := wireguard_connection.Options{
			DNSScriptDir:     nodeOptions.Directories.Script,
			HandshakeTimeout: 1 * time.Minute,
		}
		return wireguard_connection.NewConnection(opts, di.IPResolver, endpointFactory, handshakeWaiter)
	}
	di.ConnectionRegistry.Register(dvpn.ServiceType, connFactory)
}

func (di *Dependencies) bootstrapMMN() error {
	client := mmn.NewClient(di.HTTPClient, config.GetString(config.FlagMMNAPIAddress), di.SignerFactory)

	di.MMN = mmn.NewMMN(di.IPResolver, client)
	return di.MMN.Subscribe(di.EventBus)
}
