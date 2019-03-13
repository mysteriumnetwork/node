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
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/mysteriumnetwork/node/nat/traversal/config"

	"github.com/mysteriumnetwork/node/nat/traversal"

	"github.com/asaskevich/EventBus"
	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/node/blockchain"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	consumer_session "github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/consumer/statistics"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations/history"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/market"
	market_metrics "github.com/mysteriumnetwork/node/market/metrics"
	"github.com/mysteriumnetwork/node/market/metrics/oracle"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/metrics"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/nat"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
	balance_provider "github.com/mysteriumnetwork/node/session/balance/provider"
	session_payment "github.com/mysteriumnetwork/node/session/payment"
	payment_factory "github.com/mysteriumnetwork/node/session/payment/factory"
	payments_noop "github.com/mysteriumnetwork/node/session/payment/noop"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/node/session/promise/validators"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/utils"
)

// Storage stores persistent objects for future usage
type Storage interface {
	Store(issuer string, data interface{}) error
	Delete(issuer string, data interface{}) error
	Update(bucket string, object interface{}) error
	GetAllFrom(bucket string, data interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
	GetLast(bucket string, to interface{}) error
	GetBuckets() []string
	Close() error
}

// NatPinger is responsible for pinging nat holes
type NatPinger interface {
	PingProvider(ip string, port int) error
	PingTargetChan() chan json.RawMessage
	BindProvider(port int)
	BindPort(port int)
	WaitForHole() error
	Start()
}

// NatEventTracker is responsible for tracking nat events
type NatEventTracker interface {
	ConsumeNATEvent(event traversal.Event)
	LastEvent() traversal.Event
	WaitForEvent() traversal.Event
}

// Dependencies is DI container for top level components which is reused in several places
type Dependencies struct {
	Node *node.Node

	NetworkDefinition    metadata.NetworkDefinition
	MysteriumAPI         *mysterium.MysteriumAPI
	MysteriumMorqaClient market_metrics.QualityOracle
	EtherClient          *ethclient.Client

	NATService           nat.NATService
	Storage              Storage
	Keystore             *keystore.KeyStore
	PromiseStorage       *promise.Storage
	IdentityManager      identity.Manager
	SignerFactory        identity.SignerFactory
	IdentityRegistry     identity_registry.IdentityRegistry
	IdentityRegistration identity_registry.RegistrationDataProvider

	IPResolver       ip.Resolver
	LocationResolver location.Resolver
	LocationDetector location.Detector
	LocationOriginal location.Cache

	StatisticsTracker  *statistics.SessionStatisticsTracker
	StatisticsReporter *statistics.SessionStatisticsReporter
	SessionStorage     *consumer_session.Storage

	EventBus EventBus.Bus

	ConnectionManager  connection.Manager
	ConnectionRegistry *connection.Registry

	ServicesManager       *service.Manager
	ServiceRegistry       *service.Registry
	ServiceSessionStorage *session.StorageMemory

	NATPinger           NatPinger
	NATTracker          NatEventTracker
	LastSessionShutdown chan bool
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) error {
	logconfig.Bootstrap()
	nats_discovery.Bootstrap()

	log.Infof("Starting Mysterium Node (%s)", metadata.VersionAsString())

	if err := nodeOptions.Directories.Check(); err != nil {
		return err
	}

	if err := nodeOptions.Openvpn.Check(); err != nil {
		return err
	}

	if err := di.bootstrapNetworkComponents(nodeOptions.OptionsNetwork); err != nil {
		return err
	}

	if err := di.bootstrapStorage(nodeOptions.Directories.Storage); err != nil {
		return err
	}

	di.bootstrapIdentityComponents(nodeOptions)
	di.bootstrapLocationComponents(nodeOptions.Location, nodeOptions.Directories.Config)

	di.bootstrapNATComponents(nodeOptions)
	di.bootstrapServices(nodeOptions)
	di.bootstrapNodeComponents(nodeOptions)

	di.registerConnections(nodeOptions)

	err := di.subscribeEventConsumers()
	if err != nil {
		return err
	}

	if err := di.Node.Start(); err != nil {
		return err
	}

	return nil
}

func (di *Dependencies) registerOpenvpnConnection(nodeOptions node.Options) {
	service_openvpn.Bootstrap()
	connectionFactory := service_openvpn.NewProcessBasedConnectionFactory(
		// TODO instead of passing binary path here, Openvpn from node options could represent abstract vpn factory itself
		nodeOptions.Openvpn.BinaryPath(),
		nodeOptions.Directories.Config,
		nodeOptions.Directories.Runtime,
		di.LocationOriginal,
		di.SignerFactory,
		di.LastSessionShutdown,
		di.IPResolver,
		di.NATPinger,
	)
	di.ConnectionRegistry.Register(service_openvpn.ServiceType, connectionFactory)
}

func (di *Dependencies) registerNoopConnection() {
	service_noop.Bootstrap()
	di.ConnectionRegistry.Register(service_noop.ServiceType, service_noop.NewConnectionCreator())
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
	err = di.EventBus.Subscribe(traversal.EventTopic, di.NATTracker.ConsumeNATEvent)
	if err != nil {
		return err
	}

	return nil
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options) {
	dialogFactory := func(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, di.SignerFactory(consumerID))
		return dialogEstablisher.EstablishDialog(providerID, contact)
	}

	di.StatisticsTracker = statistics.NewSessionStatisticsTracker(time.Now)
	di.StatisticsReporter = statistics.NewSessionStatisticsReporter(
		di.StatisticsTracker,
		di.MysteriumAPI,
		di.SignerFactory,
		di.LocationOriginal.Get,
		time.Minute,
	)
	di.SessionStorage = consumer_session.NewSessionStorage(di.Storage, di.StatisticsTracker)
	di.PromiseStorage = promise.NewStorage(di.Storage)
	di.EventBus = EventBus.New()

	di.ConnectionRegistry = connection.NewRegistry()
	di.ConnectionManager = connection.NewManager(
		dialogFactory,
		payment_factory.PaymentIssuerFactoryFunc(nodeOptions, di.SignerFactory),
		di.ConnectionRegistry.CreateConnection,
		di.EventBus,
		di.NATPinger,
		di.IPResolver,
	)

	router := tequilapi.NewAPIRouter()
	tequilapi_endpoints.AddRouteForStop(router, utils.SoftKiller(di.Shutdown))
	tequilapi_endpoints.AddRoutesForIdentities(router, di.IdentityManager, di.SignerFactory)
	tequilapi_endpoints.AddRoutesForConnection(router, di.ConnectionManager, di.IPResolver, di.StatisticsTracker, di.MysteriumAPI)
	tequilapi_endpoints.AddRoutesForLocation(router, di.ConnectionManager, di.LocationDetector, di.LocationOriginal)
	tequilapi_endpoints.AddRoutesForProposals(router, di.MysteriumAPI, di.MysteriumMorqaClient)
	tequilapi_endpoints.AddRoutesForSession(router, di.SessionStorage)
	tequilapi_endpoints.AddRoutesForService(router, di.ServicesManager, serviceTypesRequestParser)

	identity_registry.AddIdentityRegistrationEndpoint(router, di.IdentityRegistration, di.IdentityRegistry)

	httpAPIServer := tequilapi.NewServer(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort, router)
	metricsSender := metrics.CreateSender(nodeOptions.DisableMetrics, nodeOptions.MetricsAddress)

	di.Node = node.NewNode(di.ConnectionManager, httpAPIServer, di.LocationOriginal, metricsSender, di.NATPinger.Start)
}

func newSessionManagerFactory(
	proposal market.ServiceProposal,
	sessionStorage *session.StorageMemory,
	promiseStorage session_payment.PromiseStorage,
	nodeOptions node.Options,
	natPingerChan func() chan json.RawMessage,
	lastSessionShutdown chan bool,
	natTracker NatEventTracker,
) session.ManagerFactory {
	return func(dialog communication.Dialog) *session.Manager {
		providerBalanceTrackerFactory := func(consumerID, receiverID, issuerID identity.Identity) (session.BalanceTracker, error) {
			// if the flag ain't set, just return a noop balance tracker
			if !nodeOptions.ExperimentPayments {
				return payments_noop.NewSessionBalance(), nil
			}

			timeTracker := session.NewTracker(time.Now)
			// TODO: set the time and proper payment info
			payment := dto.PaymentPerTime{
				Price: money.Money{
					Currency: money.CurrencyMyst,
					Amount:   uint64(10),
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
			tracker := balance_provider.NewBalanceTracker(&timeTracker, amountCalc, 0)
			validator := validators.NewIssuedPromiseValidator(consumerID, receiverID, issuerID)
			return session_payment.NewSessionBalance(sender, tracker, promiseChan, time.Second*5, time.Second*1, validator, promiseStorage, consumerID, receiverID, issuerID), nil
		}
		return session.NewManager(
			proposal,
			session.GenerateUUID,
			sessionStorage,
			providerBalanceTrackerFactory,
			natPingerChan,
			lastSessionShutdown,
			natTracker,
		)
	}
}

// function decides on network definition combined from testnet/localnet flags and possible overrides
func (di *Dependencies) bootstrapNetworkComponents(options node.OptionsNetwork) (err error) {
	network := metadata.DefaultNetwork

	switch {
	case options.Testnet:
		network = metadata.TestnetDefinition
	case options.Localnet:
		network = metadata.LocalnetDefinition
	}

	//override defined values one by one from options
	if options.DiscoveryAPIAddress != metadata.DefaultNetwork.DiscoveryAPIAddress {
		network.DiscoveryAPIAddress = options.DiscoveryAPIAddress
	}

	if options.BrokerAddress != metadata.DefaultNetwork.BrokerAddress {
		network.BrokerAddress = options.BrokerAddress
	}

	normalizedAddress := common.HexToAddress(options.EtherPaymentsAddress)
	if normalizedAddress != metadata.DefaultNetwork.PaymentsContractAddress {
		network.PaymentsContractAddress = normalizedAddress
	}

	if options.EtherClientRPC != metadata.DefaultNetwork.EtherClientRPC {
		network.EtherClientRPC = options.EtherClientRPC
	}

	di.NetworkDefinition = network
	di.MysteriumAPI = mysterium.NewClient(network.DiscoveryAPIAddress)
	di.MysteriumMorqaClient = oracle.NewMorqaClient(network.QualityOracle)

	log.Info("Using Eth endpoint: ", network.EtherClientRPC)
	if di.EtherClient, err = blockchain.NewClient(network.EtherClientRPC); err != nil {
		return err
	}

	log.Info("Using Eth contract at address: ", network.PaymentsContractAddress.String())
	if options.ExperimentIdentityCheck {
		if di.IdentityRegistry, err = identity_registry.NewIdentityRegistryContract(di.EtherClient, network.PaymentsContractAddress); err != nil {
			return err
		}
	} else {
		di.IdentityRegistry = &identity_registry.FakeRegistry{Registered: true, RegistrationEventExists: true}
	}

	return nil
}

func (di *Dependencies) bootstrapIdentityComponents(options node.Options) {
	di.Keystore = identity.NewKeystoreFilesystem(options.Directories.Keystore, options.Keystore.UseLightweight)
	di.IdentityManager = identity.NewIdentityManager(di.Keystore)
	di.SignerFactory = func(id identity.Identity) identity.Signer {
		return identity.NewSigner(di.Keystore, id)
	}
	di.IdentityRegistration = identity_registry.NewRegistrationDataProvider(di.Keystore)
}

func (di *Dependencies) bootstrapLocationComponents(options node.OptionsLocation, configDirectory string) {
	di.IPResolver = ip.NewResolver(options.IpifyUrl)

	switch {
	case options.Country != "":
		di.LocationResolver = location.NewStaticResolver(options.Country)
	case options.ExternalDb != "":
		di.LocationResolver = location.NewExternalDbResolver(filepath.Join(configDirectory, options.ExternalDb))
	default:
		di.LocationResolver = location.NewBuiltInResolver()
	}

	di.LocationDetector = location.NewDetector(di.IPResolver, di.LocationResolver)
	di.LocationOriginal = location.NewLocationCache(di.LocationDetector)
}

func (di *Dependencies) bootstrapNATComponents(options node.Options) error {
	if options.ExperimentNATPunching {
		di.NATTracker = traversal.NewEventsTracker()
		di.NATPinger = traversal.NewPingerFactory(di.NATTracker, config.NewConfigParser())
		di.LastSessionShutdown = make(chan bool)
	} else {
		di.NATTracker = &traversal.NoopEventsTracker{}
		di.NATPinger = &traversal.NoopPinger{}
		di.LastSessionShutdown = nil
	}
	return nil
}
