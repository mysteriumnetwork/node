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
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/consumer/migration"
	consumer_session "github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/core/beneficiary"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/monitoring"
	"github.com/mysteriumnetwork/node/core/node"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/policy/localcopy"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/state"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations/history"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrator"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/mmn"
	"github.com/mysteriumnetwork/node/nat"
	natprobe "github.com/mysteriumnetwork/node/nat/behavior"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/upnp"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/requests/resolver"
	"github.com/mysteriumnetwork/node/router"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/sleep"
	"github.com/mysteriumnetwork/node/tequilapi"
	"github.com/mysteriumnetwork/node/tequilapi/sso"
	"github.com/mysteriumnetwork/node/ui/versionmanager"
	"github.com/mysteriumnetwork/node/utils/netutil"
	paymentClient "github.com/mysteriumnetwork/payments/client"
	psort "github.com/mysteriumnetwork/payments/client/sort"
	"github.com/mysteriumnetwork/payments/observer"
)

// UIServer represents our web server
type UIServer interface {
	Serve()
	SwitchUI(path string)
	Stop()
}

// Dependencies is DI container for top level components which is reused in several places
type Dependencies struct {
	Node *Node

	HTTPTransport *http.Transport
	HTTPClient    *requests.HTTPClient

	NetworkDefinition metadata.NetworkDefinition
	MysteriumAPI      *mysterium.MysteriumAPI
	PricingHelper     *pingpong.Pricer
	EtherClientL1     *paymentClient.EthMultiClient
	EtherClientL2     *paymentClient.EthMultiClient

	SorterClientL1 *psort.MultiClientSorter
	SorterClientL2 *psort.MultiClientSorter

	EtherClients []*paymentClient.ReconnectableEthClient

	BrokerConnector  *nats.BrokerConnector
	BrokerConnection nats.Connection

	NATService       nat.NATService
	NATProber        natprobe.NATProber
	Storage          *boltdb.Bolt
	Keystore         *identity.Keystore
	IdentityManager  identity.Manager
	SignerFactory    identity.SignerFactory
	IdentityRegistry registry.IdentityRegistry
	IdentitySelector identity_selector.Handler
	IdentityMover    *identity.Mover
	FreeRegistrar    *registry.FreeRegistrar

	DiscoveryFactory    service.DiscoveryFactory
	ProposalRepository  *discovery.PricedServiceProposalRepository
	FilterPresetStorage *proposal.FilterPresetStorage
	DiscoveryWorker     discovery.Worker

	QualityClient *quality.MysteriumMORQA

	IPResolver       ip.Resolver
	LocationResolver *location.Cache

	dnsProxy *dns.Proxy

	PolicyOracle   *localcopy.Oracle
	PolicyProvider policy.Provider

	SessionStorage                   *consumer_session.Storage
	SessionConnectivityStatusStorage connectivity.StatusStorage

	EventBus eventbus.EventBus

	MultiConnectionManager     connection.MultiManager
	MultiConnectionDiagManager connection.DiagManager
	ConnectionRegistry         *connection.Registry

	ServicesManager *service.Manager
	ServiceRegistry *service.Registry
	ServiceSessions *service.SessionPool
	ServiceFirewall firewall.IncomingTrafficFirewall

	WireguardClientFactory *endpoint.WgClientFactory

	PortPool   *port.Pool
	PortMapper mapping.PortMapper

	StateKeeper *state.Keeper

	P2PDialer   p2p.Dialer
	P2PListener p2p.Listener

	Authenticator    *auth.Authenticator
	JWTAuthenticator *auth.JWTAuthenticator
	UIServer         UIServer
	Transactor       *registry.Transactor
	Affiliator       *registry.Affiliator
	BCHelper         *paymentClient.MultichainBlockchainClient
	SSOMystnodes     *sso.Mystnodes

	LogCollector *logconfig.Collector
	Reporter     *feedback.Reporter

	BeneficiarySaver    *beneficiary.Saver
	BeneficiaryProvider *beneficiary.Provider

	ProviderInvoiceStorage   *pingpong.ProviderInvoiceStorage
	ConsumerTotalsStorage    *pingpong.ConsumerTotalsStorage
	HermesPromiseStorage     *pingpong.HermesPromiseStorage
	ConsumerBalanceTracker   *pingpong.ConsumerBalanceTracker
	HermesChannelRepository  *pingpong.HermesChannelRepository
	HermesPromiseSettler     pingpong.HermesPromiseSettler
	HermesURLGetter          *pingpong.HermesURLGetter
	HermesCaller             *pingpong.HermesCaller
	HermesPromiseHandler     *pingpong.HermesPromiseHandler
	SettlementHistoryStorage *pingpong.SettlementHistoryStorage
	AddressProvider          *paymentClient.MultiChainAddressProvider
	HermesStatusChecker      *pingpong.HermesStatusChecker
	HermesMigrator           *migration.HermesMigrator

	MMN *mmn.MMN

	PilvytisAPI         *pilvytis.API
	PilvytisTracker     *pilvytis.StatusTracker
	PilvytisOrderIssuer *pilvytis.OrderIssuer

	ObserverAPI *observer.API

	ResidentCountry *identity.ResidentCountry

	BeneficiaryAddressStorage beneficiary.BeneficiaryStorage
	NodeStatusTracker         *monitoring.StatusTracker
	NodeStatsTracker          *node.StatsTracker
	uiVersionConfig           versionmanager.NodeUIVersionConfig
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) error {
	logconfig.Configure(&nodeOptions.LogOptions)

	netutil.LogNetworkStats()

	p2p.RegisterContactUnserializer()

	log.Info().Msg("Starting Mysterium Node " + metadata.VersionAsString())

	// Check early for presence of an already running node
	tequilaListener, err := di.createTequilaListener(nodeOptions)
	if err != nil {
		return err
	}

	if err := nodeOptions.Directories.Check(); err != nil {
		return err
	}

	if err := di.bootstrapFirewall(nodeOptions.Firewall); err != nil {
		return err
	}

	di.bootstrapEventBus()

	if err := di.bootstrapStorage(nodeOptions.Directories.Storage); err != nil {
		return err
	}

	if err := di.bootstrapNetworkComponents(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapLocationComponents(nodeOptions); err != nil {
		return err
	}
	if err := di.bootstrapResidentCountry(); err != nil {
		return err
	}
	if err := di.bootstrapIdentityComponents(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapDiscoveryComponents(nodeOptions.Discovery); err != nil {
		return err
	}

	if err := di.bootstrapAuthenticator(); err != nil {
		return err
	}

	if err := di.bootstrapNodeUIVersionConfig(nodeOptions); err != nil {
		return err
	}

	di.bootstrapUIServer(nodeOptions)
	if err := di.bootstrapMMN(); err != nil {
		return err
	}

	portRange, err := getUDPListenPorts()
	if err != nil {
		return err
	}

	di.PortPool = port.NewFixedRangePool(portRange)

	di.bootstrapP2P()
	di.SessionConnectivityStatusStorage = connectivity.NewStatusStorage()

	if err := di.bootstrapServices(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapQualityComponents(nodeOptions.Quality, nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapNodeComponents(nodeOptions, tequilaListener); err != nil {
		return err
	}

	di.registerConnections(nodeOptions)
	if err = di.handleConnStateChange(); err != nil {
		return err
	}

	if err := di.Node.Start(); err != nil {
		return err
	}

	config.Current.EnableEventPublishing(di.EventBus)

	di.handleNATStatusForPublicIP()

	log.Info().Msg("Mysterium node started!")
	return nil
}

func (di *Dependencies) bootstrapAddressProvider(nodeOptions node.Options) {
	ch1 := nodeOptions.Chains.Chain1
	ch2 := nodeOptions.Chains.Chain2
	chain1KnownHermeses := parseAddressSlice(ch1.KnownHermeses)
	chain2KnownHermeses := parseAddressSlice(ch2.KnownHermeses)
	hermesAddresses, err := di.ObserverAPI.GetApprovedHermesAdresses()
	if err != nil {
		log.Warn().AnErr("err", err).Msg("observer hermeses call failed, using fallback known hermeses")
	} else {
		chain1ApprovedHermeses, ok := hermesAddresses[ch1.ChainID]
		if ok {
			chain1KnownHermeses = chain1ApprovedHermeses
		}
		chain2ApprovedHermeses, ok := hermesAddresses[ch2.ChainID]
		if ok {
			chain2KnownHermeses = chain2ApprovedHermeses
		}
	}
	addresses := map[int64]paymentClient.SmartContractAddresses{
		ch1.ChainID: {
			Registry:                    common.HexToAddress(ch1.RegistryAddress),
			Myst:                        common.HexToAddress(ch1.MystAddress),
			ActiveHermes:                common.HexToAddress(ch1.HermesID),
			ActiveChannelImplementation: common.HexToAddress(ch1.ChannelImplAddress),
			KnownHermeses:               chain1KnownHermeses,
		},
		ch2.ChainID: {
			Registry:                    common.HexToAddress(ch2.RegistryAddress),
			Myst:                        common.HexToAddress(ch2.MystAddress),
			ActiveHermes:                common.HexToAddress(ch2.HermesID),
			ActiveChannelImplementation: common.HexToAddress(ch2.ChannelImplAddress),
			KnownHermeses:               chain2KnownHermeses,
		},
	}

	keeper := paymentClient.NewMultiChainAddressKeeper(addresses)
	di.AddressProvider = paymentClient.NewMultiChainAddressProvider(keeper, di.BCHelper)
}

func (di *Dependencies) bootstrapP2P() {
	verifierFactory := func(id identity.Identity) identity.Verifier {
		return identity.NewVerifierIdentity(id)
	}

	di.P2PListener = p2p.NewListener(di.BrokerConnection, di.SignerFactory, identity.NewVerifierSigned(), di.IPResolver, di.EventBus)
	di.P2PDialer = p2p.NewDialer(di.BrokerConnector, di.SignerFactory, verifierFactory, di.IPResolver, di.PortPool, di.EventBus)
}

func (di *Dependencies) createTequilaListener(nodeOptions node.Options) (net.Listener, error) {
	if !nodeOptions.TequilapiEnabled {
		return tequilapi.NewNoopListener()
	}

	tequilaListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("the port %v seems to be taken. Either you're already running a node or it is already used by another application", nodeOptions.TequilapiPort))
	}
	return tequilaListener, nil
}

func (di *Dependencies) bootstrapStateKeeper(options node.Options) error {
	deps := state.KeeperDeps{
		Publisher:                 di.EventBus,
		ServiceLister:             di.ServicesManager,
		IdentityProvider:          di.IdentityManager,
		IdentityRegistry:          di.IdentityRegistry,
		IdentityChannelCalculator: di.AddressProvider,
		BalanceProvider:           di.ConsumerBalanceTracker,
		EarningsProvider:          di.HermesChannelRepository,
		ChainID:                   options.ChainID,
		ProposalPricer:            di.ProposalRepository,
	}

	di.StateKeeper = state.NewKeeper(deps, state.DefaultDebounceDuration)
	if options.SSE.Enabled {
		return di.StateKeeper.Subscribe(di.EventBus)
	}

	return nil
}

func (di *Dependencies) registerOpenvpnConnection(nodeOptions node.Options) {
	service_openvpn.Bootstrap()
	connectionFactory := func() (connection.Connection, error) {
		return service_openvpn.NewClient(
			// TODO instead of passing binary path here, Openvpn from node options could represent abstract vpn factory itself
			nodeOptions.Openvpn.BinaryPath(),
			nodeOptions.Directories.Script,
			nodeOptions.Directories.Runtime,
			di.SignerFactory,
			di.IPResolver,
		)
	}
	di.ConnectionRegistry.Register(service_openvpn.ServiceType, connectionFactory)
}

func (di *Dependencies) registerNoopConnection() {
	service_noop.Bootstrap()
	di.ConnectionRegistry.Register(service_noop.ServiceType, service_noop.NewConnection)
}

// Shutdown stops container
func (di *Dependencies) Shutdown() (err error) {
	var errs []error
	defer func() {
		for i := range errs {
			log.Error().Err(errs[i]).Msg("Dependencies shutdown failed")
			if err == nil {
				err = errs[i]
			}
		}
	}()

	// Kill node first which includes current active VPN connection cleanup.
	if di.Node != nil {
		if err := di.Node.Kill(); err != nil {
			errs = append(errs, err)
		}
	}

	if di.ServicesManager != nil {
		if err := di.ServicesManager.Kill(); err != nil {
			errs = append(errs, err)
		}
	}

	if di.PolicyOracle != nil {
		di.PolicyOracle.Stop()
	}

	if di.NATService != nil {
		if err := di.NATService.Disable(); err != nil {
			errs = append(errs, err)
		}
	}

	if di.EtherClientL1 != nil {
		di.EtherClientL1.Close()
	}
	if di.SorterClientL1 != nil {
		di.SorterClientL1.Stop()
	}

	if di.EtherClientL2 != nil {
		di.EtherClientL2.Close()
	}
	if di.SorterClientL2 != nil {
		di.SorterClientL2.Stop()
	}

	if di.DiscoveryWorker != nil {
		di.DiscoveryWorker.Stop()
	}
	if di.PilvytisTracker != nil {
		di.PilvytisTracker.Stop()
	}
	if di.BrokerConnection != nil {
		di.BrokerConnection.Close()
	}

	if di.QualityClient != nil {
		di.QualityClient.Stop()
	}

	if di.ServiceFirewall != nil {
		di.ServiceFirewall.Teardown()
	}
	firewall.Reset()

	if di.Storage != nil {
		if err := di.Storage.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	router.Clean()

	return nil
}

func (di *Dependencies) bootstrapStorage(path string) error {
	localStorage, err := boltdb.NewStorage(path)
	if err != nil {
		return err
	}

	migrator := migrator.NewMigrator(localStorage)
	err = migrator.RunMigrations(history.Sequence)
	if err != nil {
		return err
	}

	di.Storage = localStorage

	invoiceStorage := pingpong.NewInvoiceStorage(di.Storage)
	di.ProviderInvoiceStorage = pingpong.NewProviderInvoiceStorage(invoiceStorage)
	di.ConsumerTotalsStorage = pingpong.NewConsumerTotalsStorage(di.EventBus)
	di.HermesPromiseStorage = pingpong.NewHermesPromiseStorage(di.Storage)
	di.SessionStorage = consumer_session.NewSessionStorage(di.Storage)
	di.SettlementHistoryStorage = pingpong.NewSettlementHistoryStorage(di.Storage)
	return di.SessionStorage.Subscribe(di.EventBus)
}

func (di *Dependencies) getHermesURL(nodeOptions node.Options) (string, error) {
	log.Info().Msgf("Node chain id %v", nodeOptions.ChainID)
	addr := common.HexToAddress(nodeOptions.Chains.Chain2.HermesID)
	if nodeOptions.ChainID == nodeOptions.Chains.Chain1.ChainID {
		addr = common.HexToAddress(nodeOptions.Chains.Chain1.HermesID)
	}

	hermesURL, err := di.HermesURLGetter.GetHermesURL(nodeOptions.ChainID, addr)
	if err != nil {
		return "", fmt.Errorf("could not get hermes URL: %w", err)
	}
	return hermesURL, nil
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options, tequilaListener net.Listener) error {
	di.ConsumerBalanceTracker = pingpong.NewConsumerBalanceTracker(
		di.EventBus,
		di.BCHelper,
		di.ConsumerTotalsStorage,
		di.HermesCaller,
		di.Transactor,
		di.IdentityRegistry,
		di.AddressProvider,
		di.BCHelper,
		pingpong.ConsumerBalanceTrackerConfig{
			FastSync: pingpong.PollConfig{
				Interval: nodeOptions.Payments.BalanceFastPollInterval,
				Timeout:  nodeOptions.Payments.BalanceFastPollTimeout,
			},
			LongSync: pingpong.PollConfig{
				Interval: nodeOptions.Payments.BalanceLongPollInterval,
			},
		},
	)

	err := di.ConsumerBalanceTracker.Subscribe(di.EventBus)
	if err != nil {
		return errors.Wrap(err, "could not subscribe consumer balance tracker to relevant events")
	}

	di.BeneficiaryAddressStorage = beneficiary.NewAddressStorage(di.Storage)
	di.bootstrapBeneficiaryProvider(nodeOptions)

	di.HermesPromiseHandler = pingpong.NewHermesPromiseHandler(pingpong.HermesPromiseHandlerDeps{
		HermesPromiseStorage: di.HermesPromiseStorage,
		HermesCallerFactory: func(hermesURL string) pingpong.HermesHTTPRequester {
			return pingpong.NewHermesCaller(di.HTTPClient, hermesURL)
		},
		HermesURLGetter: di.HermesURLGetter,
		FeeProvider:     di.Transactor,
		Encryption:      di.Keystore,
		EventBus:        di.EventBus,
		Signer:          di.SignerFactory,
		Chains:          []int64{nodeOptions.Chains.Chain1.ChainID, nodeOptions.Chains.Chain2.ChainID},
	})

	if err := di.HermesPromiseHandler.Subscribe(di.EventBus); err != nil {
		return err
	}

	if err := di.bootstrapHermesPromiseSettler(nodeOptions); err != nil {
		return err
	}

	di.bootstrapBeneficiarySaver(nodeOptions)

	di.ConnectionRegistry = connection.NewRegistry()

	di.MultiConnectionManager = connection.NewMultiConnectionManager(func() connection.Manager {
		return connection.NewManager(
			pingpong.ExchangeFactoryFunc(
				di.Keystore,
				di.SignerFactory,
				di.ConsumerTotalsStorage,
				di.AddressProvider,
				di.EventBus,
				nodeOptions.Payments.ConsumerDataLeewayMegabytes,
			),
			di.ConnectionRegistry.CreateConnection,
			di.EventBus,
			di.IPResolver,
			di.LocationResolver,
			connection.DefaultConfig(),
			config.GetDuration(config.FlagStatsReportInterval),
			connection.NewValidator(
				di.ConsumerBalanceTracker,
				di.IdentityManager,
			),
			di.P2PDialer,
			di.allowTrustedDomainBypassTunnel,
			di.disallowTrustedDomainBypassTunnel,
		)
	})

	if nodeOptions.ProvChecker {
		di.MultiConnectionDiagManager = connection.NewDiagManager(
			pingpong.ExchangeFactoryFunc(
				di.Keystore,
				di.SignerFactory,
				di.ConsumerTotalsStorage,
				di.AddressProvider,
				di.EventBus,
				nodeOptions.Payments.ConsumerDataLeewayMegabytes,
			),
			di.ConnectionRegistry.CreateConnection,
			di.EventBus,
			di.IPResolver,
			di.LocationResolver,
			connection.DefaultConfig(),
			config.GetDuration(config.FlagStatsReportInterval),
			connection.NewValidator(
				di.ConsumerBalanceTracker,
				di.IdentityManager,
			),
			di.P2PDialer,
			di.allowTrustedDomainBypassTunnel,
			di.disallowTrustedDomainBypassTunnel,
		)
	}
	di.NATProber = natprobe.NewNATProber(di.MultiConnectionManager, di.EventBus)

	di.LogCollector = logconfig.NewCollector(&logconfig.CurrentLogOptions)
	reporter, err := feedback.NewReporter(di.LogCollector, di.IdentityManager, di.LocationResolver, nodeOptions.FeedbackURL)
	if err != nil {
		return err
	}
	di.Reporter = reporter

	if err := di.bootstrapStateKeeper(nodeOptions); err != nil {
		return err
	}

	di.bootstrapPilvytis(nodeOptions)

	di.NodeStatusTracker = monitoring.NewStatusTracker(
		di.IdentityManager,
		di.QualityClient,
	)

	di.NodeStatsTracker = node.NewNodeStatsTracker(
		di.QualityClient.ProviderStatuses,
		di.QualityClient.ProviderSessionsList,
		di.QualityClient.ProviderTransferredData,
		di.QualityClient.ProviderSessionsCount,
		di.QualityClient.ProviderConsumersCount,
		di.QualityClient.ProviderEarningsSeries,
		di.QualityClient.ProviderSessionsSeries,
		di.QualityClient.ProviderTransferredDataSeries,
		di.QualityClient.ProviderActivityStats,
		di.QualityClient.ProviderQuality,
		di.QualityClient.ProviderServiceEarnings,
		di.IdentityManager,
	)

	di.HermesMigrator = di.bootstrapHermesMigrator()
	if err := di.HermesMigrator.Subscribe(di.EventBus); err != nil {
		return fmt.Errorf("error during subscribe: %w", err)
	}

	tequilapiHTTPServer, err := di.bootstrapTequilapi(nodeOptions, tequilaListener)
	if err != nil {
		return err
	}

	sleepNotifier := sleep.NewNotifier(di.MultiConnectionManager, di.EventBus)
	sleepNotifier.Subscribe()

	di.Node = NewNode(di.MultiConnectionManager, di.MultiConnectionDiagManager, tequilapiHTTPServer, di.EventBus, di.UIServer, sleepNotifier)

	return nil
}

// function decides on network definition combined from testnet3/localnet flags and possible overrides
func (di *Dependencies) bootstrapNetworkComponents(options node.Options) (err error) {
	optionsNetwork := options.OptionsNetwork
	network := metadata.DefaultNetwork

	switch {
	case optionsNetwork.Network.IsMainnet():
		network = metadata.MainnetDefinition
	case optionsNetwork.Network.IsLocalnet():
		network = metadata.LocalnetDefinition
	case optionsNetwork.Network.IsTestnet():
		network = metadata.TestnetDefinition
	}

	// override defined values one by one from options
	if optionsNetwork.DiscoveryAddress != metadata.DefaultNetwork.DiscoveryAddress {
		network.DiscoveryAddress = optionsNetwork.DiscoveryAddress
	}

	if !reflect.DeepEqual(optionsNetwork.BrokerAddresses, metadata.DefaultNetwork.BrokerAddresses) {
		network.BrokerAddresses = optionsNetwork.BrokerAddresses
	}

	if fmt.Sprint(optionsNetwork.EtherClientRPCL1) != fmt.Sprint(metadata.DefaultNetwork.Chain1.EtherClientRPC) {
		network.Chain1.EtherClientRPC = optionsNetwork.EtherClientRPCL1
	}
	if fmt.Sprint(optionsNetwork.EtherClientRPCL2) != fmt.Sprint(metadata.DefaultNetwork.Chain2.EtherClientRPC) {
		network.Chain2.EtherClientRPC = optionsNetwork.EtherClientRPCL2
	}

	di.NetworkDefinition = network

	dnsMap := optionsNetwork.DNSMap
	for host, hostIPs := range network.DNSMap {
		dnsMap[host] = append(dnsMap[host], hostIPs...)
	}
	for host, hostIPs := range dnsMap {
		log.Info().Msgf("Using local DNS: %s -> %s", host, hostIPs)
	}
	resolver := resolver.NewResolverMap(dnsMap)

	dialer := requests.NewDialerSwarm(options.BindAddress, options.SwarmDialerDNSHeadstart)
	dialer.ResolveContext = resolver
	di.HTTPTransport = requests.NewTransport(dialer.DialContext)
	di.HTTPClient = requests.NewHTTPClientWithTransport(di.HTTPTransport, requests.DefaultTimeout)
	di.MysteriumAPI = mysterium.NewClient(di.HTTPClient, network.DiscoveryAddress)
	di.PricingHelper = pingpong.NewPricer(di.MysteriumAPI)
	err = di.PricingHelper.Subscribe(di.EventBus)
	if err != nil {
		return err
	}

	brokerURLs := make([]*url.URL, len(di.NetworkDefinition.BrokerAddresses))
	for i, brokerAddress := range di.NetworkDefinition.BrokerAddresses {
		brokerURL, err := nats.ParseServerURL(brokerAddress)
		if err != nil {
			return err
		}
		brokerURLs[i] = brokerURL
	}

	di.BrokerConnector = nats.NewBrokerConnector(dialer.DialContext, resolver)
	if di.BrokerConnection, err = di.BrokerConnector.Connect(brokerURLs...); err != nil {
		return err
	}

	log.Info().Msgf("Using L1 Eth endpoints: %v", network.Chain1.EtherClientRPC)
	log.Info().Msgf("Using L2 Eth endpoints: %v", network.Chain2.EtherClientRPC)

	di.EtherClients = make([]*paymentClient.ReconnectableEthClient, 0)
	bcClientsL1 := make([]paymentClient.AddressableEthClientGetter, 0)
	for _, rpc := range network.Chain1.EtherClientRPC {
		client, err := paymentClient.NewReconnectableEthClient(rpc, time.Second*10)
		if err != nil {
			log.Warn().Msgf("failed to load rpc endpoint: %s", rpc)
			continue
		}
		di.EtherClients = append(di.EtherClients, client)
		bcClientsL1 = append(bcClientsL1, client)
	}

	if len(bcClientsL1) == 0 {
		log.Error().Msg("no l1 rpc endpoints loaded")
	}

	bcClientsL2 := make([]paymentClient.AddressableEthClientGetter, 0)
	for _, rpc := range network.Chain2.EtherClientRPC {
		client, err := paymentClient.NewReconnectableEthClient(rpc, time.Second*10)
		if err != nil {
			log.Warn().Msgf("failed to load rpc endpoint: %s", rpc)
			continue
		}

		di.EtherClients = append(di.EtherClients, client)
		bcClientsL2 = append(bcClientsL2, client)
	}

	if len(bcClientsL2) == 0 {
		log.Error().Msg("no l1 rpc endpoints loaded")
	}

	notifyChannelL1 := make(chan paymentClient.Notification, 5)
	di.EtherClientL1, err = paymentClient.NewEthMultiClientNotifyDown(time.Second*20, bcClientsL1, notifyChannelL1)
	if err != nil {
		return err
	}
	di.SorterClientL1 = psort.NewMultiClientSorterNoTicker(di.EtherClientL1, notifyChannelL1)
	di.SorterClientL1.AddOnNotificationAction(psort.DefaultByAvailability)
	go di.SorterClientL1.Run()

	notifyChannelL2 := make(chan paymentClient.Notification, 5)
	di.EtherClientL2, err = paymentClient.NewEthMultiClientNotifyDown(time.Second*20, bcClientsL2, notifyChannelL2)
	if err != nil {
		return err
	}
	di.SorterClientL2 = psort.NewMultiClientSorterNoTicker(di.EtherClientL2, notifyChannelL2)
	di.SorterClientL2.AddOnNotificationAction(psort.DefaultByAvailability)
	go di.SorterClientL2.Run()

	bcL1 := paymentClient.NewBlockchain(di.EtherClientL1, options.Payments.BCTimeout)
	bcL2 := paymentClient.NewBlockchain(di.EtherClientL2, options.Payments.BCTimeout)

	clients := make(map[int64]paymentClient.BC)
	clients[options.Chains.Chain1.ChainID] = bcL1
	clients[options.Chains.Chain2.ChainID] = bcL2

	di.BCHelper = paymentClient.NewMultichainBlockchainClient(clients)
	di.ObserverAPI = observer.NewAPI(options.ObserverAddress, time.Second*30)
	di.bootstrapAddressProvider(options)
	di.HermesURLGetter = pingpong.NewHermesURLGetter(di.BCHelper, di.AddressProvider, di.ObserverAPI)

	registryStorage := registry.NewRegistrationStatusStorage(di.Storage)

	hermesURL, err := di.getHermesURL(options)
	if err != nil {
		return err
	}

	di.HermesCaller = pingpong.NewHermesCaller(di.HTTPClient, hermesURL)
	di.SignerFactory = func(id identity.Identity) identity.Signer {
		return identity.NewSigner(di.Keystore, id)
	}

	if err := di.bootstrapSSOMystnodes(); err != nil {
		return err
	}

	di.Transactor = registry.NewTransactor(
		di.HTTPClient,
		options.Transactor.TransactorEndpointAddress,
		di.AddressProvider,
		di.SignerFactory,
		di.EventBus,
		di.BCHelper,
		options.Transactor.TransactorFeesValidTime,
	)
	di.Affiliator = registry.NewAffiliator(di.HTTPClient, options.Affiliator.AffiliatorEndpointAddress)

	registryCfg := registry.IdentityRegistryConfig{
		TransactorPollInterval: options.Payments.RegistryTransactorPollInterval,
		TransactorPollTimeout:  options.Payments.RegistryTransactorPollTimeout,
	}

	if di.IdentityRegistry, err = registry.NewIdentityRegistryContract(di.EtherClientL2, di.AddressProvider, registryStorage, di.EventBus, di.HermesCaller, di.Transactor, di.IdentitySelector, registryCfg); err != nil {
		return err
	}

	allow := []string{
		network.DiscoveryAddress,
		options.Transactor.TransactorEndpointAddress,
		options.Affiliator.AffiliatorEndpointAddress,
		hermesURL,
		options.PilvytisAddress,
	}
	allow = append(allow, network.Chain1.EtherClientRPC...)
	allow = append(allow, network.Chain2.EtherClientRPC...)

	if err := di.AllowURLAccess(allow...); err != nil {
		return err
	}

	di.WireguardClientFactory = endpoint.NewWGClientFactory()

	return di.IdentityRegistry.Subscribe(di.EventBus)
}

func (di *Dependencies) bootstrapEventBus() {
	di.EventBus = eventbus.New()
}

func (di *Dependencies) bootstrapIdentityComponents(options node.Options) error {
	var ks *keystore.KeyStore
	if options.Keystore.UseLightweight {
		log.Debug().Msg("Using lightweight keystore")
		ks = keystore.NewKeyStore(options.Directories.Keystore, keystore.LightScryptN, keystore.LightScryptP)
	} else {
		log.Debug().Msg("Using heavyweight keystore")
		ks = keystore.NewKeyStore(options.Directories.Keystore, keystore.StandardScryptN, keystore.StandardScryptP)
	}

	di.Keystore = identity.NewKeystoreFilesystem(options.Directories.Keystore, ks)
	if di.ResidentCountry == nil {
		return errMissingDependency("di.residentCountry")
	}
	di.IdentityManager = identity.NewIdentityManager(di.Keystore, di.EventBus, di.ResidentCountry)

	di.IdentitySelector = identity_selector.NewHandler(
		di.IdentityManager,
		identity.NewIdentityCache(options.Directories.Keystore, "remember.json"),
		di.SignerFactory,
	)
	di.IdentityMover = identity.NewMover(
		di.Keystore,
		di.EventBus,
		di.SignerFactory)

	di.FreeRegistrar = registry.NewFreeRegistrar(di.IdentitySelector, di.Transactor, di.IdentityRegistry, options.Transactor.TryFreeRegistration)
	if err := di.FreeRegistrar.Subscribe(di.EventBus); err != nil {
		return err
	}
	return nil
}

func (di *Dependencies) bootstrapQualityComponents(options node.OptionsQuality, nodeOptions node.Options) (err error) {
	if err := di.AllowURLAccess(options.Address); err != nil {
		return err
	}

	di.QualityClient = quality.NewMorqaClient(
		requests.NewHTTPClientWithTransport(di.HTTPTransport, 60*time.Second),
		options.Address,
		di.SignerFactory,
	)
	go di.QualityClient.Start()

	var transport quality.Transport
	switch options.Type {
	case node.QualityTypeElastic:
		transport = quality.NewElasticSearchTransport(di.HTTPClient, options.Address, 10*time.Second)
	case node.QualityTypeMORQA:
		transport = quality.NewMORQATransport(di.QualityClient, di.LocationResolver)
	case node.QualityTypeNone:
		transport = quality.NewNoopTransport()
	default:
		err = errors.Errorf("unknown Quality Oracle provider: %s", options.Type)
	}
	if err != nil {
		return err
	}

	// Quality metrics
	qualitySender := quality.NewSender(transport, metadata.VersionAsString())
	if err := qualitySender.Subscribe(di.EventBus); err != nil {
		return err
	}

	// warm up the loader as the load takes up to a couple of secs
	loader := &upnp.GatewayLoader{}
	go loader.Get()
	natSender := event.NewSender(qualitySender, di.IPResolver.GetPublicIP, loader.HumanReadable)
	if err := natSender.Subscribe(di.EventBus); err != nil {
		return err
	}

	return nil
}

func (di *Dependencies) bootstrapLocationComponents(options node.Options) (err error) {
	if err = di.AllowURLAccess(options.Location.IPDetectorURL); err != nil {
		return errors.Wrap(err, "failed to add firewall exception")
	}

	ipResolver := ip.NewResolver(di.HTTPClient, options.BindAddress, options.Location.IPDetectorURL, ip.IPFallbackAddresses)
	di.IPResolver = ip.NewCachedResolver(ipResolver, 5*time.Minute)

	var resolver location.Resolver
	switch options.Location.Type {
	case node.LocationTypeManual:
		resolver = location.NewStaticResolver(options.Location.Country, options.Location.City, options.Location.IPType, di.IPResolver)
	case node.LocationTypeBuiltin:
		resolver, err = location.NewBuiltInResolver(di.IPResolver)
	case node.LocationTypeMMDB:
		resolver, err = location.NewExternalDBResolver(filepath.Join(options.Directories.Script, options.Location.Address), di.IPResolver)
	case node.LocationTypeOracle:
		if err := di.AllowURLAccess(options.Location.Address); err != nil {
			return err
		}
		resolver, err = location.NewOracleResolver(di.HTTPClient, options.Location.Address), nil
	default:
		err = errors.Errorf("unknown location provider: %s", options.Location.Type)
	}
	if err != nil {
		return err
	}

	di.LocationResolver = location.NewCache(resolver, di.EventBus, time.Minute*5)

	if !config.GetBool(config.FlagProxyMode) && !config.GetBool(config.FlagDVPNMode) {
		err = di.EventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, di.LocationResolver.HandleConnectionEvent)
		if err != nil {
			return err
		}
	}

	err = di.EventBus.SubscribeAsync(nodevent.AppTopicNode, di.LocationResolver.HandleNodeEvent)
	if err != nil {
		return err
	}

	return nil
}

func (di *Dependencies) bootstrapAuthenticator() error {
	key, err := auth.NewJWTEncryptionKey(di.Storage)
	if err != nil {
		return err
	}
	di.Authenticator = auth.NewAuthenticator()
	di.JWTAuthenticator = auth.NewJWTAuthenticator(key)

	return nil
}

func (di *Dependencies) bootstrapPilvytis(options node.Options) {
	di.PilvytisAPI = pilvytis.NewAPI(di.HTTPClient, options.PilvytisAddress, di.SignerFactory, di.LocationResolver, di.AddressProvider)
	di.PilvytisTracker = pilvytis.NewStatusTracker(di.PilvytisAPI, di.IdentityManager, di.EventBus, time.Minute)
	di.PilvytisOrderIssuer = pilvytis.NewOrderIssuer(di.PilvytisAPI, di.PilvytisTracker)

	go di.PilvytisTracker.Track()
	di.PilvytisTracker.SubscribeAsync(di.EventBus)
}

func (di *Dependencies) bootstrapFirewall(options node.OptionsFirewall) error {
	firewall.DefaultOutgoingFirewall = firewall.NewOutgoingTrafficFirewall(config.GetBool(config.FlagOutgoingFirewall))
	if err := firewall.DefaultOutgoingFirewall.Setup(); err != nil {
		return err
	}

	di.ServiceFirewall = firewall.NewIncomingTrafficFirewall(config.GetBool(config.FlagIncomingFirewall))
	if err := di.ServiceFirewall.Setup(); err != nil {
		return err
	}

	if options.BlockAlways {
		bindAddress := "0.0.0.0"
		resolver := ip.NewResolver(di.HTTPClient, bindAddress, "", ip.IPFallbackAddresses)
		outboundIP, err := resolver.GetOutboundIP()
		if err != nil {
			return err
		}

		_, err = firewall.BlockNonTunnelTraffic(firewall.Global, outboundIP)
		return err
	}
	return nil
}

func (di *Dependencies) bootstrapBeneficiaryProvider(options node.Options) {
	di.BeneficiaryProvider = beneficiary.NewProvider(
		options.ChainID,
		di.AddressProvider,
		di.Storage,
		di.BCHelper,
	)
}

func (di *Dependencies) bootstrapBeneficiarySaver(options node.Options) {
	di.BeneficiarySaver = beneficiary.NewSaver(
		options.ChainID,
		di.AddressProvider,
		di.Storage,
		di.BCHelper,
		di.HermesPromiseSettler,
	)
}

func (di *Dependencies) bootstrapHermesMigrator() *migration.HermesMigrator {
	return migration.NewHermesMigrator(
		di.Transactor,
		di.AddressProvider,
		di.HermesURLGetter,
		func(hermesURL string) pingpong.HermesHTTPRequester {
			return pingpong.NewHermesCaller(di.HTTPClient, hermesURL)
		},
		di.HermesPromiseSettler,
		di.IdentityRegistry,
		di.ConsumerBalanceTracker,
		migration.NewStorage(di.Storage, di.AddressProvider),
		di.BCHelper,
	)
}

func (di *Dependencies) bootstrapSSOMystnodes() error {
	s := sso.NewMystnodes(di.SignerFactory, di.HTTPClient)
	di.SSOMystnodes = s
	return s.Subscribe(di.EventBus)
}

func (di *Dependencies) handleConnStateChange() error {
	if di.HTTPClient == nil {
		return errors.New("HTTPClient is not initialized")
	}

	latestState := connectionstate.NotConnected
	return di.EventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, func(e connectionstate.AppEventConnectionState) {
		if config.GetBool(config.FlagProxyMode) || config.GetBool(config.FlagDVPNMode) || config.GetBool(config.FlagProvCheckerMode) {
			return // Proxy mode doesn't establish system wide tunnels, no reconnect required.
		}

		// Here we care only about connected and disconnected events.
		if e.State != connectionstate.Connected && e.State != connectionstate.NotConnected {
			return
		}

		isDisconnected := latestState == connectionstate.Connected && e.State == connectionstate.NotConnected
		isConnected := latestState == connectionstate.NotConnected && e.State == connectionstate.Connected
		if isDisconnected || isConnected {
			netutil.LogNetworkStats()

			log.Info().Msg("Reconnecting HTTP clients due to VPN connection state change")
			di.HTTPTransport.CloseIdleConnections()

			for _, cl := range di.EtherClients {
				if err := cl.Reconnect(time.Second * 15); err != nil {
					log.Warn().Err(err).Msg("Ethereum client failed to reconnect, will retry one more time")
					// Default golang DNS resolver does not allow to reload /etc/resolv.conf more than once per 5 seconds.
					// This could lead to the problem, when right after connect/disconnect new DNS config not applied instantly.
					// Doing a couple of retries here to make sure we reconnected Ethererum client correctly.
					// Default DNS timeout is 10 seconds. It's enough to try to reconnect only twice to cover 5 seconds lag for DNS config reload.
					// https://github.com/mysteriumnetwork/node/issues/2282
					if err := cl.Reconnect(time.Second * 15); err != nil {
						log.Error().Err(err).Msg("Ethereum client failed to reconnect")
					}
				}
			}

			di.EventBus.Publish(registry.AppTopicEthereumClientReconnected, struct{}{})
		}
		latestState = e.State
	})
}

func (di *Dependencies) handleNATStatusForPublicIP() {
	outIP, err := di.IPResolver.GetOutboundIP()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get outbound IP address")
	}

	pubIP, err := di.IPResolver.GetPublicIP()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get public IP address")
	}

	if outIP == pubIP && pubIP != "" {
		di.EventBus.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent("", "public_ip"))
	}
}

func (di *Dependencies) bootstrapResidentCountry() error {
	if di.EventBus == nil {
		return errMissingDependency("di.EventBus")
	}

	if di.LocationResolver == nil {
		return errMissingDependency("di.LocationResolver")
	}
	di.ResidentCountry = identity.NewResidentCountry(di.EventBus, di.LocationResolver)
	return nil
}

func errMissingDependency(dep string) error {
	return errors.New("Missing dependency: " + dep)
}

// AllowURLAccess allows the requested addresses to be served when the tunnel is active.
func (di *Dependencies) AllowURLAccess(servers ...string) error {
	if _, err := firewall.AllowURLAccess(servers...); err != nil {
		return err
	}

	if _, err := di.ServiceFirewall.AllowURLAccess(servers...); err != nil {
		return err
	}

	return nil
}

func getUDPListenPorts() (port.Range, error) {
	udpPortRange, err := port.ParseRange(config.GetString(config.FlagUDPListenPorts))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse UDP listen port range, using default value")
		return port.Range{}, fmt.Errorf("failed to parse UDP ports: %w", err)
	}
	return udpPortRange, nil
}

func (di *Dependencies) allowTrustedDomainBypassTunnel() {
	allow := []string{di.NetworkDefinition.DiscoveryAddress}
	allow = append(allow, di.NetworkDefinition.BrokerAddresses...)

	if err := router.ExcludeURL(allow...); err != nil {
		log.Error().Err(err).Msgf("Failed to exclude routes for trusted domains: %v", allow)
	}
}

func (di *Dependencies) disallowTrustedDomainBypassTunnel() {
	allow := []string{di.NetworkDefinition.DiscoveryAddress}
	allow = append(allow, di.NetworkDefinition.BrokerAddresses...)

	if err := router.RemoveExcludedURL(allow...); err != nil {
		log.Error().Err(err).Msgf("Failed to remove excluded routes for trusted domains: %v", allow)
	}
}

func parseAddressSlice(slice []string) []common.Address {
	res := make([]common.Address, len(slice))
	for i, s := range slice {
		res[i] = common.HexToAddress(s)
	}
	return res
}
