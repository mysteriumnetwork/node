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
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	appconfig "github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	consumer_session "github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/consumer/statistics"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery"
	discovery_api "github.com/mysteriumnetwork/node/core/discovery/api"
	discovery_broker "github.com/mysteriumnetwork/node/core/discovery/broker"
	discovery_composite "github.com/mysteriumnetwork/node/core/discovery/composite"
	discovery_noop "github.com/mysteriumnetwork/node/core/discovery/noop"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/state"
	statevent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations/history"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/firewall/vnd"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/nat/traversal/config"
	"github.com/mysteriumnetwork/node/nat/upnp"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/services"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/connectivity"
	sessionevent "github.com/mysteriumnetwork/node/session/event"
	session_payment "github.com/mysteriumnetwork/node/session/payment"
	payment_factory "github.com/mysteriumnetwork/node/session/payment/factory"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/node/session/promise/validators"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/tequilapi/sse"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NatPinger is responsible for pinging nat holes
type NatPinger interface {
	PingProvider(ip string, port int, consumerPort int, stop <-chan struct{}) error
	PingTarget(*traversal.Params)
	BindServicePort(serviceType services.ServiceType, port int)
	Start()
	Stop()
	SetProtectSocketCallback(SocketProtect func(socket int) bool)
	StopNATProxy()
	Valid() bool
}

// UIServer represents our web server
type UIServer interface {
	Serve() error
	Stop()
}

// Dependencies is DI container for top level components which is reused in several places
type Dependencies struct {
	Node *node.Node

	HTTPClient *requests.HTTPClient

	NetworkDefinition metadata.NetworkDefinition
	MysteriumAPI      *mysterium.MysteriumAPI
	EtherClient       *ethclient.Client

	BrokerConnector  *nats.BrokerConnector
	BrokerConnection nats.Connection

	NATService       nat.NATService
	Storage          *boltdb.Bolt
	Keystore         *keystore.KeyStore
	PromiseStorage   *promise.Storage
	IdentityManager  identity.Manager
	SignerFactory    identity.SignerFactory
	IdentityRegistry identity_registry.IdentityRegistry
	IdentitySelector identity_selector.Handler

	DiscoveryFactory service.DiscoveryFactory
	DiscoveryStorage *discovery.ProposalStorage
	DiscoveryFinder  discovery.ProposalFinder

	QualityMetricsSender *quality.Sender
	QualityClient        *quality.MysteriumMORQA

	IPResolver       ip.Resolver
	LocationResolver *location.Cache

	StatisticsTracker                *statistics.SessionStatisticsTracker
	StatisticsReporter               *statistics.SessionStatisticsReporter
	SessionStorage                   *consumer_session.Storage
	SessionConnectivityStatusStorage connectivity.StatusStorage

	EventBus eventbus.EventBus

	ConnectionManager  connection.Manager
	ConnectionRegistry *connection.Registry

	ServicesManager       *service.Manager
	ServiceRegistry       *service.Registry
	ServiceSessionStorage *session.EventBasedStorage

	NATPinger      NatPinger
	NATTracker     *event.Tracker
	NATEventSender *event.Sender

	BandwidthTracker *bandwidth.Tracker

	StateKeeper *state.Keeper

	Authenticator     *auth.Authenticator
	JWTAuthenticator  *auth.JWTAuthenticator
	UIServer          UIServer
	SSEHandler        *sse.Handler
	Transactor        *registry.Transactor
	BCHelper          *pingpong.BlockchainWithRetries
	ProviderRegistrar *registry.ProviderRegistrar

	LogCollector *logconfig.Collector
	Reporter     *feedback.Reporter

	ProviderInvoiceStorage   *pingpong.ProviderInvoiceStorage
	ConsumerInvoiceStorage   *pingpong.ConsumerInvoiceStorage
	ConsumerTotalsStorage    *pingpong.ConsumerTotalsStorage
	AccountantPromiseStorage *pingpong.AccountantPromiseStorage
	ConsumerBalanceTracker   *pingpong.ConsumerBalanceTracker
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) error {
	logconfig.Configure(&nodeOptions.LogOptions)
	nats_discovery.Bootstrap()
	di.BrokerConnector = nats.NewBrokerConnector()

	log.Info().Msg("Starting Mysterium Node " + metadata.VersionAsString())

	di.HTTPClient = requests.NewHTTPClient(nodeOptions.BindAddress, requests.DefaultTimeout)

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

	if err := di.bootstrapStorage(nodeOptions.Directories.Storage); err != nil {
		return err
	}

	di.bootstrapEventBus()

	if err := di.bootstrapNetworkComponents(nodeOptions); err != nil {
		return err
	}

	di.bootstrapIdentityComponents(nodeOptions)

	if err := di.bootstrapDiscoveryComponents(nodeOptions.Discovery); err != nil {
		return err
	}
	if err := di.bootstrapLocationComponents(nodeOptions); err != nil {
		return err
	}
	if err := di.bootstrapAuthenticator(); err != nil {
		return err
	}

	di.bootstrapUIServer(nodeOptions)
	di.bootstrapMMN(nodeOptions)
	if err := di.bootstrapBandwidthTracker(); err != nil {
		return err
	}

	if err := di.bootstrapServices(nodeOptions); err != nil {
		return err
	}

	di.bootstrapNATComponents(nodeOptions)

	if err := di.bootstrapStateKeeper(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapSSEHandler(); err != nil {
		return err
	}

	if err := di.bootstrapQualityComponents(nodeOptions.BindAddress, nodeOptions.Quality); err != nil {
		return err
	}

	di.bootstrapNodeComponents(nodeOptions, tequilaListener)
	if err := di.bootstrapProviderRegistrar(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapAccountantPromiseSettler(nodeOptions); err != nil {
		return err
	}

	di.registerConnections(nodeOptions)

	if err = di.subscribeEventConsumers(); err != nil {
		return err
	}
	if err = di.DiscoveryFinder.Start(); err != nil {
		return err
	}
	if err := di.Node.Start(); err != nil {
		return err
	}

	appconfig.Current.EnableEventPublishing(di.EventBus)

	log.Info().Msg("Mysterium node started!")
	return nil
}

func (di *Dependencies) createTequilaListener(nodeOptions node.Options) (net.Listener, error) {
	if !nodeOptions.TequilapiEnabled {
		return tequilapi.NewNoopListener()
	}

	tequilaListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("The port %v seems to be taken. Either you're already running a node or it is already used by another application", nodeOptions.TequilapiPort))
	}
	return tequilaListener, nil
}

func (di *Dependencies) bootstrapStateKeeper(options node.Options) error {
	var lastStageName string
	if options.ExperimentNATPunching {
		lastStageName = traversal.StageName
	} else {
		lastStageName = mapping.StageName
	}

	tracker := nat.NewStatusTracker(lastStageName)

	di.StateKeeper = state.NewKeeper(tracker, di.EventBus, di.ServicesManager, di.ServiceSessionStorage, state.DefaultDebounceDuration)

	err := di.EventBus.SubscribeAsync(service.StatusTopic, di.StateKeeper.ConsumeServiceStateEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.SubscribeAsync(sessionevent.Topic, di.StateKeeper.ConsumeSessionStateEvent)
	if err != nil {
		return err
	}
	return di.EventBus.SubscribeAsync(event.Topic, di.StateKeeper.ConsumeNATEvent)
}

func (di *Dependencies) registerOpenvpnConnection(nodeOptions node.Options) {
	service_openvpn.Bootstrap()
	connectionFactory := service_openvpn.NewProcessBasedConnectionFactory(
		// TODO instead of passing binary path here, Openvpn from node options could represent abstract vpn factory itself
		nodeOptions.Openvpn.BinaryPath(),
		nodeOptions.Directories.Config,
		nodeOptions.Directories.Runtime,
		di.SignerFactory,
		di.IPResolver,
		di.NATPinger,
	)
	di.ConnectionRegistry.Register(service_openvpn.ServiceType, connectionFactory)
}

func (di *Dependencies) registerNoopConnection() {
	service_noop.Bootstrap()
	di.ConnectionRegistry.Register(service_noop.ServiceType, service_noop.NewConnectionCreator())
}

// bootstrapSSEHandler bootstraps the SSEHandler and all of its dependencies
func (di *Dependencies) bootstrapSSEHandler() error {
	di.SSEHandler = sse.NewHandler(di.StateKeeper)
	err := di.EventBus.Subscribe(nodevent.Topic, di.SSEHandler.ConsumeNodeEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.Subscribe(statevent.Topic, di.SSEHandler.ConsumeStateEvent)
	return err
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

	if di.ServicesManager != nil {
		if err := di.ServicesManager.Kill(); err != nil {
			errs = append(errs, err)
		}
	}

	if di.NATService != nil {
		if err := di.NATService.Disable(); err != nil {
			errs = append(errs, err)
		}
	}
	if di.DiscoveryFinder != nil {
		di.DiscoveryFinder.Stop()
	}
	if di.Storage != nil {
		if err := di.Storage.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if di.BrokerConnection != nil {
		di.BrokerConnection.Close()
	}

	firewall.Reset()

	if di.Node != nil {
		if err := di.Node.Kill(); err != nil {
			errs = append(errs, err)
		}
	}
	return nil
}

func (di *Dependencies) bootstrapStorage(path string) error {
	localStorage, err := boltdb.NewStorage(path)
	if err != nil {
		return err
	}

	migrator := boltdb.NewMigrator(localStorage)
	err = migrator.RunMigrations(history.Sequence)
	if err != nil {
		return err
	}

	di.Storage = localStorage

	invoiceStorage := pingpong.NewInvoiceStorage(di.Storage)
	di.ProviderInvoiceStorage = pingpong.NewProviderInvoiceStorage(invoiceStorage)
	di.ConsumerInvoiceStorage = pingpong.NewConsumerInvoiceStorage(invoiceStorage)
	di.ConsumerTotalsStorage = pingpong.NewConsumerTotalsStorage(di.Storage)
	di.AccountantPromiseStorage = pingpong.NewAccountantPromiseStorage(di.Storage)
	return nil
}

func (di *Dependencies) subscribeEventConsumers() error {
	// state events
	err := di.EventBus.Subscribe(connection.SessionEventTopic, di.StatisticsTracker.ConsumeSessionEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.Subscribe(connection.SessionEventTopic, di.StatisticsReporter.ConsumeSessionEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.Subscribe(connection.SessionEventTopic, di.SessionStorage.ConsumeSessionEvent)
	if err != nil {
		return err
	}

	// statistics events
	err = di.EventBus.Subscribe(connection.StatisticsEventTopic, di.StatisticsTracker.ConsumeStatisticsEvent)
	if err != nil {
		return err
	}

	// NAT events
	err = di.EventBus.Subscribe(event.Topic, di.NATEventSender.ConsumeNATEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.Subscribe(event.Topic, di.NATTracker.ConsumeNATEvent)
	if err != nil {
		return err
	}

	// Quality metrics
	err = di.EventBus.SubscribeAsync(connection.StateEventTopic, di.QualityMetricsSender.SendConnStateEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.SubscribeAsync(connection.SessionEventTopic, di.QualityMetricsSender.SendSessionEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.SubscribeAsync(connection.StatisticsEventTopic, di.QualityMetricsSender.SendSessionData)
	if err != nil {
		return err
	}
	err = di.EventBus.SubscribeAsync(discovery.ProposalEventTopic, di.QualityMetricsSender.SendProposalEvent)
	if err != nil {
		return err
	}

	err = di.handleHTTPClientConnections()
	if err != nil {
		return err
	}

	return di.EventBus.SubscribeAsync(nodevent.Topic, di.QualityMetricsSender.SendStartupEvent)
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options, tequilaListener net.Listener) error {
	dialogFactory := func(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, di.SignerFactory(consumerID), di.BrokerConnector)
		return dialogEstablisher.EstablishDialog(providerID, contact)
	}

	di.StatisticsTracker = statistics.NewSessionStatisticsTracker(time.Now)
	di.StatisticsReporter = statistics.NewSessionStatisticsReporter(
		di.StatisticsTracker,
		di.MysteriumAPI,
		di.SignerFactory,
		di.LocationResolver,
		time.Minute,
	)
	di.SessionStorage = consumer_session.NewSessionStorage(di.Storage, di.StatisticsTracker)
	di.PromiseStorage = promise.NewStorage(di.Storage)
	di.SessionConnectivityStatusStorage = connectivity.NewStatusStorage()

	channelImplementation := nodeOptions.Transactor.ChannelImplementation

	di.Transactor = registry.NewTransactor(
		di.HTTPClient,
		nodeOptions.Transactor.TransactorEndpointAddress,
		nodeOptions.Transactor.RegistryAddress,
		nodeOptions.Accountant.AccountantID,
		channelImplementation,
		di.SignerFactory,
		di.EventBus,
	)

	di.ConsumerBalanceTracker = pingpong.NewConsumerBalanceTracker(
		di.EventBus,
		common.HexToAddress(nodeOptions.Payments.MystSCAddress),
		di.BCHelper,
		pingpong.NewChannelAddressCalculator(
			nodeOptions.Accountant.AccountantID,
			nodeOptions.Transactor.ChannelImplementation,
			nodeOptions.Transactor.RegistryAddress,
		),
		pingpong.NewAccountantCaller(requests.NewHTTPClient(nodeOptions.BindAddress, time.Second*5), nodeOptions.Accountant.AccountantEndpointAddress).GetConsumerData,
	)

	err := di.ConsumerBalanceTracker.Subscribe(di.EventBus)
	if err != nil {
		return errors.Wrap(err, "could not subscribe consumer balance tracker to relevant events")
	}

	di.ConnectionRegistry = connection.NewRegistry()
	di.ConnectionManager = connection.NewManager(
		dialogFactory,
		pingpong.BackwardsCompatibleExchangeFactoryFunc(
			di.Keystore,
			nodeOptions,
			di.SignerFactory,
			di.ConsumerInvoiceStorage,
			di.ConsumerTotalsStorage,
			nodeOptions.Transactor.ChannelImplementation,
			nodeOptions.Transactor.RegistryAddress,
			di.EventBus),
		di.ConnectionRegistry.CreateConnection,
		di.EventBus,
		connectivity.NewStatusSender(),
		di.IPResolver,
		connection.DefaultIPCheckParams(),
		nodeOptions.Payments.PaymentsDisabled,
	)

	di.LogCollector = logconfig.NewCollector(&logconfig.CurrentLogOptions)
	reporter, err := feedback.NewReporter(di.LogCollector, di.IdentityManager, nodeOptions.FeedbackURL)
	if err != nil {
		return err
	}
	di.Reporter = reporter

	tequilapiHTTPServer := di.bootstrapTequilapi(nodeOptions, tequilaListener, channelImplementation)

	di.Node = node.NewNode(di.ConnectionManager, tequilapiHTTPServer, di.EventBus, di.NATPinger, di.UIServer)
	return nil
}

func (di *Dependencies) bootstrapTequilapi(nodeOptions node.Options, listener net.Listener, channelImplementation string) tequilapi.APIServer {
	if !nodeOptions.TequilapiEnabled {
		return tequilapi.NewNoopAPIServer()
	}

	router := tequilapi.NewAPIRouter()
	tequilapi_endpoints.AddRouteForStop(router, utils.SoftKiller(di.Shutdown))
	tequilapi_endpoints.AddRoutesForAuthentication(router, di.Authenticator, di.JWTAuthenticator)
	tequilapi_endpoints.AddRoutesForIdentities(router, di.IdentityManager, di.IdentitySelector, di.IdentityRegistry, nodeOptions.Transactor.RegistryAddress, channelImplementation, di.ConsumerBalanceTracker.GetBalance)
	tequilapi_endpoints.AddRoutesForConnection(router, di.ConnectionManager, di.StatisticsTracker, di.DiscoveryStorage, di.IdentityRegistry)
	tequilapi_endpoints.AddRoutesForConnectionSessions(router, di.SessionStorage)
	tequilapi_endpoints.AddRoutesForConnectionLocation(router, di.ConnectionManager, di.IPResolver, di.LocationResolver, di.LocationResolver)
	tequilapi_endpoints.AddRoutesForProposals(router, di.DiscoveryStorage, di.QualityClient)
	tequilapi_endpoints.AddRoutesForService(router, di.ServicesManager, serviceTypesRequestParser, nodeOptions.AccessPolicyEndpointAddress)
	tequilapi_endpoints.AddRoutesForServiceSessions(router, di.StateKeeper)
	tequilapi_endpoints.AddRoutesForPayout(router, di.IdentityManager, di.SignerFactory, di.MysteriumAPI)
	tequilapi_endpoints.AddRoutesForAccessPolicies(di.HTTPClient, router, nodeOptions.AccessPolicyEndpointAddress)
	tequilapi_endpoints.AddRoutesForNAT(router, di.StateKeeper.GetState)
	tequilapi_endpoints.AddRoutesForSSE(router, di.SSEHandler)
	tequilapi_endpoints.AddRoutesForTransactor(router, di.Transactor)
	tequilapi_endpoints.AddRoutesForConfig(router)
	tequilapi_endpoints.AddRoutesForFeedback(router, di.Reporter)
	tequilapi_endpoints.AddRoutesForConnectivityStatus(router, di.SessionConnectivityStatusStorage)
	identity_registry.AddIdentityRegistrationEndpoint(router, di.IdentityRegistry)
	corsPolicy := tequilapi.NewMysteriumCorsPolicy()
	return tequilapi.NewServer(listener, router, corsPolicy)
}

func newSessionManagerFactory(
	nodeOptions node.Options,
	proposal market.ServiceProposal,
	sessionStorage *session.EventBasedStorage,
	providerInvoiceStorage *pingpong.ProviderInvoiceStorage,
	consumerInvoiceStorage *pingpong.ConsumerInvoiceStorage,
	accountantPromiseStorage *pingpong.AccountantPromiseStorage,
	promiseStorage session_payment.PromiseStorage,
	natPingerChan func(*traversal.Params),
	natTracker *event.Tracker,
	serviceID string,
	eventbus eventbus.EventBus,
	bcHelper *pingpong.BlockchainWithRetries,
	transactor *registry.Transactor,
	paymentsDisabled bool,
) session.ManagerFactory {
	return func(dialog communication.Dialog) *session.Manager {
		providerBalanceTrackerFactory := func(consumerID, receiverID, issuerID identity.Identity) (session.PaymentEngine, error) {
			timeTracker := session.NewTracker(time.Now)
			// TODO: set the time and proper payment info
			payment := dto.PaymentRate{
				Price: money.Money{
					Currency: money.CurrencyMyst,
					Amount:   uint64(0),
				},
				Duration: time.Minute,
			}
			amountCalc := session.AmountCalc{PaymentDef: payment}
			sender := balance.NewBalanceSender(dialog)
			promiseChan := make(chan promise.Message, 1)
			listener := promise.NewListener(promiseChan)
			err := dialog.Receive(listener.GetConsumer())
			if err != nil {
				return nil, err
			}

			// TODO: the ints and times here need to be passed in as well, or defined as constants
			tracker := balance.NewBalanceTracker(&timeTracker, amountCalc, 0)
			validator := validators.NewIssuedPromiseValidator(consumerID, receiverID, issuerID)
			return session_payment.NewSessionBalance(sender, tracker, promiseChan, payment_factory.BalanceSendPeriod, payment_factory.PromiseWaitTimeout, validator, promiseStorage, consumerID, receiverID, issuerID), nil
		}

		paymentEngineFactory := pingpong.InvoiceFactoryCreator(
			dialog, payment_factory.BalanceSendPeriod,
			payment_factory.PromiseWaitTimeout, providerInvoiceStorage,
			pingpong.DefaultPaymentInfo,
			pingpong.NewAccountantCaller(requests.NewHTTPClient(nodeOptions.BindAddress, time.Second*5), nodeOptions.Accountant.AccountantEndpointAddress),
			accountantPromiseStorage,
			nodeOptions.Transactor.RegistryAddress,
			nodeOptions.Transactor.ChannelImplementation,
			pingpong.DefaultAccountantFailureCount,
			uint16(nodeOptions.Payments.MaxAllowedPaymentPercentile),
			nodeOptions.Payments.MaxRRecoveryLength,
			bcHelper,
			eventbus,
			transactor,
		)
		return session.NewManager(
			proposal,
			session.GenerateUUID,
			sessionStorage,
			providerBalanceTrackerFactory,
			paymentEngineFactory,
			natPingerChan,
			natTracker,
			serviceID,
			eventbus,
			paymentsDisabled,
		)
	}
}

// function decides on network definition combined from testnet/localnet flags and possible overrides
func (di *Dependencies) bootstrapNetworkComponents(options node.Options) (err error) {
	optionsNetwork := options.OptionsNetwork
	network := metadata.DefaultNetwork

	switch {
	case optionsNetwork.Testnet:
		network = metadata.TestnetDefinition
	case optionsNetwork.Localnet:
		network = metadata.LocalnetDefinition
	}

	//override defined values one by one from options
	if optionsNetwork.MysteriumAPIAddress != metadata.DefaultNetwork.MysteriumAPIAddress {
		network.MysteriumAPIAddress = optionsNetwork.MysteriumAPIAddress
	}

	if optionsNetwork.QualityOracle != metadata.DefaultNetwork.QualityOracle {
		network.QualityOracle = optionsNetwork.QualityOracle
	}

	if optionsNetwork.BrokerAddress != metadata.DefaultNetwork.BrokerAddress {
		network.BrokerAddress = optionsNetwork.BrokerAddress
	}

	if optionsNetwork.EtherClientRPC != metadata.DefaultNetwork.EtherClientRPC {
		network.EtherClientRPC = optionsNetwork.EtherClientRPC
	}

	di.NetworkDefinition = network

	if _, err := firewall.AllowURLAccess(
		network.EtherClientRPC,
		network.MysteriumAPIAddress,
		options.Transactor.TransactorEndpointAddress,
	); err != nil {
		return err
	}

	di.MysteriumAPI = mysterium.NewClient(di.HTTPClient, network.MysteriumAPIAddress)

	if di.BrokerConnection, err = di.BrokerConnector.Connect(di.NetworkDefinition.BrokerAddress); err != nil {
		return err
	}

	log.Info().Msg("Using Eth endpoint: " + network.EtherClientRPC)
	if di.EtherClient, err = ethclient.Dial(network.EtherClientRPC); err != nil {
		return err
	}

	bc := pingpong.NewBlockchain(di.EtherClient, options.Payments.BCTimeout)
	di.BCHelper = pingpong.NewBlockchainWithRetries(bc, time.Millisecond*300, 3)

	registryStorage := registry.NewRegistrationStatusStorage(di.Storage)
	if di.IdentityRegistry, err = identity_registry.NewIdentityRegistryContract(di.EtherClient, common.HexToAddress(options.Transactor.RegistryAddress), common.HexToAddress(options.Accountant.AccountantID), registryStorage, di.EventBus); err != nil {
		return err
	}

	return di.IdentityRegistry.Subscribe(di.EventBus)
}

func (di *Dependencies) bootstrapEventBus() {
	di.EventBus = eventbus.New()
}

func (di *Dependencies) bootstrapIdentityComponents(options node.Options) {
	di.Keystore = identity.NewKeystoreFilesystem(options.Directories.Keystore, options.Keystore.UseLightweight)
	di.IdentityManager = identity.NewIdentityManager(di.Keystore, di.EventBus)
	di.SignerFactory = func(id identity.Identity) identity.Signer {
		return identity.NewSigner(di.Keystore, id)
	}
	di.IdentitySelector = identity_selector.NewHandler(
		di.IdentityManager,
		di.MysteriumAPI,
		identity.NewIdentityCache(options.Directories.Keystore, "remember.json"),
		di.SignerFactory,
	)

}

func (di *Dependencies) bootstrapDiscoveryComponents(options node.OptionsDiscovery) error {
	di.DiscoveryStorage = discovery.NewStorage()

	discoveryRegistry := discovery_composite.NewRegistry()
	discoveryFinder := discovery_composite.NewFinder()
	for _, discoveryType := range options.Types {
		switch discoveryType {
		case node.DiscoveryTypeAPI:
			discoveryRegistry.AddRegistry(
				discovery_api.NewRegistry(di.MysteriumAPI),
			)

			if !options.ProposalFetcherEnabled {
				discoveryFinder.AddFinder(discovery_noop.NewFinder())
			} else {
				discoveryFinder.AddFinder(
					discovery_api.NewFinder(di.DiscoveryStorage, di.MysteriumAPI.Proposals, 30*time.Minute),
				)
			}
		case node.DiscoveryTypeBroker:
			discoveryRegistry.AddRegistry(
				discovery_broker.NewRegistry(di.BrokerConnection),
			)
			discoveryFinder.AddFinder(
				discovery_broker.NewFinder(di.DiscoveryStorage, di.BrokerConnection, options.PingInterval+time.Second, 1*time.Second),
			)
		default:
			return errors.Errorf("unknown discovery adapter: %s", discoveryType)
		}
	}

	di.DiscoveryFinder = discoveryFinder
	di.DiscoveryFactory = func() service.Discovery {
		return discovery.NewService(di.IdentityRegistry, discoveryRegistry, options.PingInterval, di.SignerFactory, di.EventBus)
	}
	return nil
}

func (di *Dependencies) bootstrapQualityComponents(bindAddress string, options node.OptionsQuality) (err error) {
	if _, err := firewall.AllowURLAccess(di.NetworkDefinition.QualityOracle); err != nil {
		return err
	}
	di.QualityClient = quality.NewMorqaClient(bindAddress, options.Address, 20*time.Second)

	var transport quality.Transport
	switch options.Type {
	case node.QualityTypeElastic:
		_, err = firewall.AllowURLAccess(options.Address)
		transport = quality.NewElasticSearchTransport(di.HTTPClient, options.Address, 10*time.Second)
	case node.QualityTypeMORQA:
		_, err = firewall.AllowURLAccess(options.Address)
		transport = quality.NewMORQATransport(di.QualityClient)
	case node.QualityTypeNone:
		transport = quality.NewNoopTransport()
	default:
		err = errors.Errorf("unknown Quality Oracle provider: %s", options.Type)
	}
	if err != nil {
		return err
	}
	di.QualityMetricsSender = quality.NewSender(transport, metadata.VersionAsString(), di.ConnectionManager, di.LocationResolver)

	// warm up the loader as the load takes up to a couple of secs
	loader := &upnp.GatewayLoader{}
	go loader.Get()
	di.NATEventSender = event.NewSender(di.QualityMetricsSender, di.IPResolver.GetPublicIP, loader.HumanReadable)
	return nil
}

func (di *Dependencies) bootstrapLocationComponents(options node.Options) (err error) {
	if _, err = firewall.AllowURLAccess(options.Location.IPDetectorURL); err != nil {
		return errors.Wrap(err, "failed to add firewall exception")
	}
	di.IPResolver = ip.NewResolver(di.HTTPClient, options.BindAddress, options.Location.IPDetectorURL)

	var resolver location.Resolver
	switch options.Location.Type {
	case node.LocationTypeManual:
		resolver = location.NewStaticResolver(options.Location.Country, options.Location.City, options.Location.NodeType, di.IPResolver)
	case node.LocationTypeBuiltin:
		resolver, err = location.NewBuiltInResolver(di.IPResolver)
	case node.LocationTypeMMDB:
		resolver, err = location.NewExternalDBResolver(filepath.Join(options.Directories.Config, options.Location.Address), di.IPResolver)
	case node.LocationTypeOracle:
		if _, err := firewall.AllowURLAccess(options.Location.Address); err != nil {
			return err
		}
		resolver, err = location.NewOracleResolver(di.HTTPClient, options.Location.Address), nil
	default:
		err = errors.Errorf("unknown location provider: %s", options.Location.Type)
	}
	if err != nil {
		return err
	}

	di.LocationResolver = location.NewCache(resolver, time.Minute*5)

	err = di.EventBus.SubscribeAsync(connection.StateEventTopic, di.LocationResolver.HandleConnectionEvent)
	if err != nil {
		return err
	}

	err = di.EventBus.SubscribeAsync(nodevent.Topic, di.LocationResolver.HandleNodeEvent)
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
	di.Authenticator = auth.NewAuthenticator(di.Storage)
	di.JWTAuthenticator = auth.NewJWTAuthenticator(key)

	return nil
}

func (di *Dependencies) bootstrapBandwidthTracker() error {
	di.BandwidthTracker = &bandwidth.Tracker{}
	err := di.EventBus.SubscribeAsync(connection.SessionEventTopic, di.BandwidthTracker.ConsumeSessionEvent)
	if err != nil {
		return err
	}

	return di.EventBus.SubscribeAsync(connection.StatisticsEventTopic, di.BandwidthTracker.ConsumeStatisticsEvent)
}

func (di *Dependencies) bootstrapNATComponents(options node.Options) {
	di.NATTracker = event.NewTracker()
	if options.ExperimentNATPunching {
		log.Debug().Msg("Experimental NAT punching enabled, creating a pinger")
		di.NATPinger = traversal.NewPinger(
			di.NATTracker,
			config.NewConfigParser(),
			traversal.NewNATProxy(),
			mapping.StageName,
			di.EventBus,
		)
	} else {
		di.NATPinger = &traversal.NoopPinger{}
	}
}

func (di *Dependencies) bootstrapFirewall(options node.OptionsFirewall) error {
	fwVendor, err := vnd.SetupVendor()
	if err != nil {
		return err
	}
	firewall.Configure(fwVendor)
	if options.BlockAlways {
		_, err := firewall.BlockNonTunnelTraffic(firewall.Global)
		return err
	}
	return nil
}

func (di *Dependencies) handleHTTPClientConnections() error {
	if di.HTTPClient == nil {
		return errors.New("HTTPClient is not initialized")
	}

	latestState := connection.NotConnected
	return di.EventBus.Subscribe(connection.StateEventTopic, func(e connection.StateEvent) {
		// Here we care only about connected and disconnected events.
		if e.State != connection.Connected && e.State != connection.NotConnected {
			return
		}

		isDisconnected := latestState == connection.Connected && e.State == connection.NotConnected
		isConnected := latestState == connection.NotConnected && e.State == connection.Connected
		if isDisconnected || isConnected {
			log.Info().Msg("Reconnecting HTTP clients due to VPN connection state change")
			di.HTTPClient.Reconnect()
			di.QualityClient.Reconnect()
			di.BrokerConnector.ReconnectAll()
		}
		latestState = e.State
	})
}
