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
	"path/filepath"
	"time"

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
	promise_noop "github.com/mysteriumnetwork/node/core/promise/methods/noop"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations/history"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/server/metrics"
	"github.com/mysteriumnetwork/node/server/metrics/oracle"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
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
	Close() error
}

// Dependencies is DI container for top level components which is reusedin several places
type Dependencies struct {
	Node *node.Node

	NetworkDefinition    metadata.NetworkDefinition
	MysteriumClient      server.Client
	MysteriumMorqaClient metrics.QualityOracle
	EtherClient          *ethclient.Client

	Storage              Storage
	Keystore             *keystore.KeyStore
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

	ServiceRunner         *service.Runner
	ServiceRegistry       *service.Registry
	ServiceSessionStorage *session.StorageMemory
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

	di.bootstrapIdentityComponents(nodeOptions.Directories)
	di.bootstrapLocationComponents(nodeOptions.Location, nodeOptions.Directories.Config)
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
	)
	di.ConnectionRegistry.Register(service_openvpn.ServiceType, connectionFactory.CreateConnection)
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

	if di.ServiceRunner != nil {
		runnerErrs := di.ServiceRunner.KillAll()
		errs = append(errs, runnerErrs...)
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
	err := di.EventBus.Subscribe(connection.StateEventTopic, di.StatisticsTracker.ConsumeStateEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.Subscribe(connection.StateEventTopic, di.StatisticsReporter.ConsumeStateEvent)
	if err != nil {
		return err
	}
	err = di.EventBus.Subscribe(connection.StateEventTopic, di.SessionStorage.ConsumeStateEvent)
	if err != nil {
		return err
	}

	// statistics events
	err = di.EventBus.Subscribe(connection.StatisticsEventTopic, di.StatisticsTracker.ConsumeStatisticsEvent)
	if err != nil {
		return err
	}

	return nil
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options) {
	dialogFactory := func(consumerID, providerID identity.Identity, contact dto_discovery.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, di.SignerFactory(consumerID))
		return dialogEstablisher.EstablishDialog(providerID, contact)
	}

	promiseIssuerFactory := func(issuerID identity.Identity, dialog communication.Dialog) connection.PromiseIssuer {
		if nodeOptions.ExperimentPromiseCheck {
			return promise_noop.NewPromiseIssuer(issuerID, dialog, di.SignerFactory(issuerID))
		}
		return &promise_noop.FakePromiseEngine{}
	}

	di.StatisticsTracker = statistics.NewSessionStatisticsTracker(time.Now)
	di.StatisticsReporter = statistics.NewSessionStatisticsReporter(
		di.StatisticsTracker,
		di.MysteriumClient,
		di.SignerFactory,
		di.LocationOriginal.Get,
		time.Minute,
	)
	di.SessionStorage = consumer_session.NewSessionStorage(di.Storage, di.StatisticsTracker)

	di.EventBus = EventBus.New()

	di.ConnectionRegistry = connection.NewRegistry()
	di.ConnectionManager = connection.NewManager(
		dialogFactory,
		promiseIssuerFactory,
		di.ConnectionRegistry.CreateConnection,
		di.EventBus,
	)

	router := tequilapi.NewAPIRouter()
	tequilapi_endpoints.AddRouteForStop(router, utils.SoftKiller(di.Shutdown))
	tequilapi_endpoints.AddRoutesForIdentities(router, di.IdentityManager, di.MysteriumClient, di.SignerFactory)
	tequilapi_endpoints.AddRoutesForConnection(router, di.ConnectionManager, di.IPResolver, di.StatisticsTracker, di.MysteriumClient)
	tequilapi_endpoints.AddRoutesForLocation(router, di.ConnectionManager, di.LocationDetector, di.LocationOriginal)
	tequilapi_endpoints.AddRoutesForProposals(router, di.MysteriumClient, di.MysteriumMorqaClient)
	tequilapi_endpoints.AddRoutesForSession(router, di.SessionStorage)
	identity_registry.AddIdentityRegistrationEndpoint(router, di.IdentityRegistration, di.IdentityRegistry)

	httpAPIServer := tequilapi.NewServer(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort, router)

	di.Node = node.NewNode(di.ConnectionManager, httpAPIServer, di.LocationOriginal)
}

func newSessionManagerFactory(
	proposal dto_discovery.ServiceProposal,
	configProvider session.ConfigProvider,
	sessionStorage *session.StorageMemory,
	promiseHandler func(dialog communication.Dialog) session.PromiseProcessor,
) session.ManagerFactory {
	return func(dialog communication.Dialog) session.Manager {
		return session.NewManager(
			proposal,
			session.GenerateUUID,
			configProvider,
			sessionStorage,
			promiseHandler(dialog),
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
	di.MysteriumClient = server.NewClient(network.DiscoveryAPIAddress)
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

func (di *Dependencies) bootstrapIdentityComponents(directories node.OptionsDirectory) {
	di.Keystore = identity.NewKeystoreFilesystem(directories.Keystore)
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
