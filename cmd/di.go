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
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	consumer_session "github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/consumer/statistics"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery"
	discovery_api "github.com/mysteriumnetwork/node/core/discovery/api"
	discovery_broker "github.com/mysteriumnetwork/node/core/discovery/broker"
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
	"github.com/mysteriumnetwork/node/core/transactor"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/firewall/vnd"
	"github.com/mysteriumnetwork/node/identity"
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
	"github.com/mysteriumnetwork/node/services"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
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
)

// Storage stores persistent objects for future usage
type Storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
	Store(issuer string, data interface{}) error
	Delete(issuer string, data interface{}) error
	Update(bucket string, object interface{}) error
	GetAllFrom(bucket string, data interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
	GetLast(bucket string, to interface{}) error
	GetBuckets() []string
	Close() error
}

type serviceSessionStorage interface {
	Add(sessionInstance session.Session)
	GetAll() []session.Session
	UpdateDataTransfer(id session.ID, up, down int64)
	Find(id session.ID) (session.Session, bool)
	Remove(id session.ID)
	RemoveForService(serviceID string)
}

// JWTAuthenticator provides authentication for Tequilapi and UI
type JWTAuthenticator interface {
	CreateToken(username string) (token auth.JWT, err error)
	ValidateToken(token string) (bool, error)
}

// Authenticator provides authentication for Tequilapi and UI
type Authenticator interface {
	CheckCredentials(username, password string) error
	ChangePassword(username, oldPassword, newPassword string) error
}

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

// NatEventTracker is responsible for tracking NAT events
type NatEventTracker interface {
	ConsumeNATEvent(event event.Event)
	LastEvent() *event.Event
	WaitForEvent() event.Event
}

// NatEventSender is responsible for sending NAT events to metrics server
type NatEventSender interface {
	ConsumeNATEvent(event event.Event)
}

// CacheResolver caches the location resolution results
type CacheResolver interface {
	location.Resolver
	location.OriginResolver
	HandleNodeEvent(se nodevent.Payload)
	HandleConnectionEvent(connection.StateEvent)
}

// UIServer represents our web server
type UIServer interface {
	Serve() error
	Stop()
}

// Transactor represents interface to Transactor service
type Transactor interface {
	FetchFees() (transactor.Fees, error)
	RegisterIdentity(identity string, regReqDTO *transactor.IdentityRegistrationRequestDTO) error
}

// Dependencies is DI container for top level components which is reused in several places
type Dependencies struct {
	Node *node.Node

	NetworkDefinition metadata.NetworkDefinition
	MysteriumAPI      *mysterium.MysteriumAPI
	EtherClient       *ethclient.Client

	NATService       nat.NATService
	Storage          Storage
	Keystore         *keystore.KeyStore
	PromiseStorage   *promise.Storage
	IdentityManager  identity.Manager
	SignerFactory    identity.SignerFactory
	IdentityRegistry identity_registry.IdentityRegistry
	IdentitySelector identity_selector.Handler

	DiscoveryFactory    service.DiscoveryFactory
	DiscoveryFinder     *discovery.Finder
	DiscoveryFetcherAPI *discovery_api.Fetcher

	QualityMetricsSender *quality.Sender
	QualityClient        *quality.MysteriumMORQA

	IPResolver       ip.Resolver
	LocationResolver CacheResolver

	StatisticsTracker  *statistics.SessionStatisticsTracker
	StatisticsReporter *statistics.SessionStatisticsReporter
	SessionStorage     *consumer_session.Storage

	EventBus eventbus.EventBus

	ConnectionManager  connection.Manager
	ConnectionRegistry *connection.Registry

	ServicesManager       *service.Manager
	ServiceRegistry       *service.Registry
	ServiceSessionStorage serviceSessionStorage

	NATPinger      NatPinger
	NATTracker     NatEventTracker
	NATEventSender NatEventSender

	BandwidthTracker *bandwidth.Tracker

	StateKeeper *state.Keeper

	Authenticator    Authenticator
	JWTAuthenticator JWTAuthenticator
	UIServer         UIServer
	SSEHandler       *sse.Handler
	Transactor       Transactor
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) error {
	logconfig.BootstrapWith(&nodeOptions.LogOptions)
	nats_discovery.Bootstrap()

	log.Infof("Starting Mysterium Node (%s)", metadata.VersionAsString())
	log.Infof("Build information (%s)", metadata.BuildAsString())

	// check early for presence of an already running node
	tequilaListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("The port %v seems to be taken. Either you're already running a node or it is already used by another application", nodeOptions.TequilapiPort))
	}

	if err := nodeOptions.Directories.Check(); err != nil {
		return err
	}

	if err := di.bootstrapFirewall(nodeOptions.Firewall); err != nil {
		return err
	}

	if err := di.bootstrapNetworkComponents(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapStorage(nodeOptions.Directories.Storage); err != nil {
		return err
	}

	di.bootstrapEventBus()
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

	if err := di.bootstrapBandwidthTracker(); err != nil {
		return err
	}

	if err := di.bootstrapSSEHandler(); err != nil {
		return err
	}

	di.bootstrapNATComponents(nodeOptions)
	if err := di.bootstrapServices(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapStateKeeper(nodeOptions); err != nil {
		return err
	}

	if err := di.bootstrapQualityComponents(nodeOptions.BindAddress, nodeOptions.Quality); err != nil {
		return err
	}

	di.bootstrapNodeComponents(nodeOptions, tequilaListener)

	di.registerConnections(nodeOptions)

	if err = di.subscribeEventConsumers(); err != nil {
		return err
	}
	if err = di.DiscoveryFetcherAPI.Start(); err != nil {
		return err
	}
	if err := di.Node.Start(); err != nil {
		return err
	}

	return nil
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
			log.Error("Dependencies shutdown failed: ", errs[i])
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
	if di.DiscoveryFetcherAPI != nil {
		di.DiscoveryFetcherAPI.Stop()
	}
	if di.Node != nil {
		if err := di.Node.Kill(); err != nil {
			errs = append(errs, err)
		}
	}
	if di.Storage != nil {
		if err := di.Storage.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	firewall.Reset()
	log.Flush()
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
	err = di.EventBus.SubscribeAsync(connection.StateEventTopic, di.QualityMetricsSender.SendSessionEvent)
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
	return di.EventBus.SubscribeAsync(nodevent.Topic, di.QualityMetricsSender.SendStartupEvent)
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options, listener net.Listener) {
	dialogFactory := func(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, di.SignerFactory(consumerID))
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

	di.ConnectionRegistry = connection.NewRegistry()
	di.ConnectionManager = connection.NewManager(
		dialogFactory,
		payment_factory.PaymentIssuerFactoryFunc(nodeOptions, di.SignerFactory),
		pingpong.ExchangeMessageFactoryCreator(di.Keystore),
		di.ConnectionRegistry.CreateConnection,
		di.EventBus,
	)

	di.Transactor = transactor.NewTransactor(
		nodeOptions.BindAddress,
		nodeOptions.Transactor.TransactorEndpointAddress,
		nodeOptions.Transactor.RegistryAddress,
		nodeOptions.Transactor.AccountantID,
		di.SignerFactory,
	)

	router := tequilapi.NewAPIRouter()
	tequilapi_endpoints.AddRouteForStop(router, utils.SoftKiller(di.Shutdown))
	tequilapi_endpoints.AddRoutesForAuthentication(router, di.Authenticator, di.JWTAuthenticator)
	tequilapi_endpoints.AddRoutesForIdentities(router, di.IdentityManager, di.IdentitySelector, di.IdentityRegistry)
	tequilapi_endpoints.AddRoutesForConnection(router, di.ConnectionManager, di.StatisticsTracker, di.DiscoveryFinder)
	tequilapi_endpoints.AddRoutesForConnectionSessions(router, di.SessionStorage)
	tequilapi_endpoints.AddRoutesForConnectionLocation(router, di.ConnectionManager, di.IPResolver, di.LocationResolver, di.LocationResolver)
	tequilapi_endpoints.AddRoutesForProposals(router, di.DiscoveryFinder, di.QualityClient)
	tequilapi_endpoints.AddRoutesForService(router, di.ServicesManager, serviceTypesRequestParser, nodeOptions.AccessPolicyEndpointAddress)
	tequilapi_endpoints.AddRoutesForServiceSessions(router, di.StateKeeper)
	tequilapi_endpoints.AddRoutesForPayout(router, di.IdentityManager, di.SignerFactory, di.MysteriumAPI)
	tequilapi_endpoints.AddRoutesForAccessPolicies(nodeOptions.BindAddress, router, nodeOptions.AccessPolicyEndpointAddress)
	tequilapi_endpoints.AddRoutesForNAT(router, di.StateKeeper.GetState)
	tequilapi_endpoints.AddRoutesForSSE(router, di.SSEHandler)
	tequilapi_endpoints.AddRoutesForTransactor(router, di.Transactor)
	tequilapi_endpoints.AddRoutesForConfig(router)

	identity_registry.AddIdentityRegistrationEndpoint(router, di.IdentityRegistry)

	corsPolicy := tequilapi.NewMysteriumCorsPolicy()
	httpAPIServer := tequilapi.NewServer(listener, router, corsPolicy)

	di.Node = node.NewNode(di.ConnectionManager, httpAPIServer, di.EventBus, di.NATPinger, di.UIServer)
}

func newSessionManagerFactory(
	proposal market.ServiceProposal,
	sessionStorage serviceSessionStorage,
	promiseStorage session_payment.PromiseStorage,
	natPingerChan func(*traversal.Params),
	natTracker NatEventTracker,
	serviceID string,
	eventbus eventbus.EventBus,
) session.ManagerFactory {
	return func(dialog communication.Dialog) *session.Manager {
		providerBalanceTrackerFactory := func(consumerID, receiverID, issuerID identity.Identity) (session.PaymentEngine, error) {
			timeTracker := session.NewTracker(time.Now)
			// TODO: set the time and proper payment info
			payment := dto.PaymentPerTime{
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

		paymentEngineFactory := pingpong.InvoiceFactoryCreator(dialog, payment_factory.BalanceSendPeriod, payment_factory.PromiseWaitTimeout)
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

	normalizedAddress := common.HexToAddress(optionsNetwork.EtherPaymentsAddress)
	if normalizedAddress != metadata.DefaultNetwork.PaymentsContractAddress {
		network.PaymentsContractAddress = normalizedAddress
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

	di.MysteriumAPI = mysterium.NewClient(options.BindAddress, network.MysteriumAPIAddress)

	log.Info("Using Eth endpoint: ", network.EtherClientRPC)

	if di.EtherClient, err = ethclient.Dial(network.EtherClientRPC); err != nil {
		return err
	}

	log.Info("Using Eth contract at address: ", network.PaymentsContractAddress.String())
	log.Info("options.ExperimentIdentityCheck: ", optionsNetwork.ExperimentIdentityCheck)
	if optionsNetwork.ExperimentIdentityCheck {
		if di.IdentityRegistry, err = identity_registry.NewIdentityRegistryContract(di.EtherClient, network.PaymentsContractAddress, common.HexToAddress(options.Transactor.AccountantID)); err != nil {
			return err
		}
	} else {
		di.IdentityRegistry = &identity_registry.FakeRegistry{Registered: true, RegistrationEventExists: true}
	}

	return nil
}

func (di *Dependencies) bootstrapEventBus() {
	di.EventBus = eventbus.New()
}

func (di *Dependencies) bootstrapIdentityComponents(options node.Options) {
	di.Keystore = identity.NewKeystoreFilesystem(options.Directories.Keystore, options.Keystore.UseLightweight)
	di.IdentityManager = identity.NewIdentityManager(di.Keystore)
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
	var registry discovery.ProposalRegistry
	switch options.Type {
	case node.DiscoveryTypeAPI:
		registry = discovery_api.NewRegistry(di.MysteriumAPI)
	case node.DiscoveryTypeBroker:
		sender := discovery_broker.NewSender(nats.NewConnectionMock())
		registry = discovery_broker.NewRegistry(sender)
	default:
		return errors.Errorf("unknown discovery provider: %s", options.Type)
	}

	di.DiscoveryFactory = func() service.Discovery {
		return discovery.NewService(di.IdentityRegistry, registry, di.SignerFactory, di.EventBus)
	}

	storage := discovery.NewStorage()
	di.DiscoveryFinder = discovery.NewFinder(storage)
	di.DiscoveryFetcherAPI = discovery_api.NewFetcher(storage, di.MysteriumAPI.Proposals, 30*time.Second)

	return nil
}

func (di *Dependencies) bootstrapQualityComponents(bindAddress string, options node.OptionsQuality) (err error) {
	if _, err := firewall.AllowURLAccess(di.NetworkDefinition.QualityOracle); err != nil {
		return err
	}
	di.QualityClient = quality.NewMorqaClient(options.Address, 20*time.Second)

	var transport quality.Transport
	switch options.Type {
	case node.QualityTypeElastic:
		_, err = firewall.AllowURLAccess(options.Address)
		transport = quality.NewElasticSearchTransport(bindAddress, options.Address, 10*time.Second)
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
	di.IPResolver = ip.NewResolver(options.BindAddress, options.Location.IPDetectorURL)

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
		resolver, err = location.NewOracleResolver(options.BindAddress, options.Location.Address), nil
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
		log.Trace("experimental NAT punching enabled, creating a pinger")
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
