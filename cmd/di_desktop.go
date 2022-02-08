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

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/mmn"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/p2p"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/mysteriumnetwork/node/session/pingpong"
	pingpong_noop "github.com/mysteriumnetwork/node/session/pingpong/noop"
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

	di.bootstrapServiceOpenvpn(nodeOptions)
	di.bootstrapServiceNoop(nodeOptions)
	di.bootstrapServiceWireguard(nodeOptions)

	return nil
}

func (di *Dependencies) bootstrapServiceWireguard(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		wireguard.ServiceType,
		func(serviceOptions service.Options) (service.Service, error) {
			loc, err := di.LocationResolver.DetectLocation()
			if err != nil {
				return nil, err
			}

			wgOptions := serviceOptions.(wireguard_service.Options)

			svc := wireguard_service.NewManager(
				di.IPResolver,
				loc.Country,
				di.NATService,
				di.EventBus,
				wgOptions,
				di.PortPool,
				di.ServiceFirewall,
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

func (di *Dependencies) bootstrapProviderRegistrar(nodeOptions node.Options) error {
	if nodeOptions.Consumer {
		log.Debug().Msg("Skipping provider registrar for consumer mode")
		return nil
	}

	cfg := registry.ProviderRegistrarConfig{
		DelayBetweenRetries: nodeOptions.Transactor.ProviderRegistrationRetryDelay,
	}

	di.ProviderRegistrar = registry.NewProviderRegistrar(di.Transactor, di.IdentityRegistry, di.AddressProvider, di.BCHelper, cfg)
	return di.ProviderRegistrar.Subscribe(di.EventBus)
}

func (di *Dependencies) bootstrapHermesPromiseSettler(nodeOptions node.Options) error {
	di.HermesChannelRepository = pingpong.NewHermesChannelRepository(
		di.HermesPromiseStorage,
		di.BCHelper,
		di.EventBus,
		di.BeneficiaryProvider,
	)

	if err := di.HermesChannelRepository.Subscribe(di.EventBus); err != nil {
		log.Error().Err(err).Msg("Failed to subscribe channel repository")
		return errors.Wrap(err, "could not subscribe channel repository to relevant events")
	}

	if nodeOptions.Consumer {
		log.Debug().Msg("Skipping hermes promise settler for consumer mode")
		di.HermesPromiseSettler = &pingpong_noop.NoopHermesPromiseSettler{}
		return nil
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
		pingpong.HermesPromiseSettlerConfig{
			Threshold:                    nodeOptions.Payments.HermesPromiseSettlingThreshold,
			SettlementCheckTimeout:       nodeOptions.Payments.SettlementTimeout,
			SettlementCheckInterval:      nodeOptions.Payments.SettlementRecheckInterval,
			L1ChainID:                    nodeOptions.Chains.Chain1.ChainID,
			L2ChainID:                    nodeOptions.Chains.Chain2.ChainID,
			ZeroStakeSettlementThreshold: nodeOptions.Payments.ZeroStakeSettlementThreshold,
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

	di.PolicyOracle = policy.NewOracle(
		di.HTTPClient,
		config.GetString(config.FlagAccessPolicyAddress),
		config.GetDuration(config.FlagAccessPolicyFetchInterval),
	)
	go di.PolicyOracle.Start()

	di.HermesStatusChecker = pingpong.NewHermesStatusChecker(di.BCHelper, nodeOptions.Payments.HermesStatusRecheckInterval)

	newP2PSessionHandler := func(serviceInstance *service.Instance, channel p2p.Channel) *service.SessionManager {
		paymentEngineFactory := pingpong.InvoiceFactoryCreator(
			channel, nodeOptions.Payments.ProviderInvoiceFrequency,
			pingpong.PromiseWaitTimeout, di.ProviderInvoiceStorage,
			pingpong.DefaultHermesFailureCount,
			uint16(nodeOptions.Payments.MaxAllowedPaymentPercentile),
			nodeOptions.Payments.MaxUnpaidInvoiceValue,
			di.HermesStatusChecker,
			di.EventBus,
			di.HermesPromiseHandler,
			di.AddressProvider,
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
	di.registerWireguardConnection(nodeOptions)
}

func (di *Dependencies) registerWireguardConnection(nodeOptions node.Options) {
	wireguard.Bootstrap()
	handshakeWaiter := wireguard_connection.NewHandshakeWaiter()
	endpointFactory := func() (wireguard.ConnectionEndpoint, error) {
		resourceAllocator := resources.NewAllocator(nil, wireguard_service.DefaultOptions.Subnet)
		return endpoint.NewConnectionEndpoint(resourceAllocator)
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

func (di *Dependencies) bootstrapMMN() error {
	client := mmn.NewClient(di.HTTPClient, config.GetString(config.FlagMMNAPIAddress), di.SignerFactory)

	di.MMN = mmn.NewMMN(di.IPResolver, client)
	return di.MMN.Subscribe(di.EventBus)
}
