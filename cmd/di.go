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

	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/node/blockchain"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/promise/methods/noop"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/server"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/utils"
)

// Dependencies is DI container for top level components which is reusedin several places
type Dependencies struct {
	NodeOptions node.Options
	Node        *node.Node

	NetworkDefinition metadata.NetworkDefinition
	MysteriumClient   server.Client
	EtherClient       *ethclient.Client

	Keystore             *keystore.KeyStore
	IdentityManager      identity.Manager
	SignerFactory        identity.SignerFactory
	IdentityRegistry     identity_registry.IdentityRegistry
	IdentityRegistration identity_registry.RegistrationDataProvider

	IPResolver       ip.Resolver
	LocationResolver location.Resolver
	LocationDetector location.Detector
	LocationOriginal location.Cache

	StatsKeeper stats.SessionStatsKeeper

	ConnectionManager  connection.Manager
	ConnectionCreators map[string]connection.ConnectionCreator
	ServiceManager     *service.Manager
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) error {
	logconfig.Bootstrap()
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

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

	di.bootstrapIdentityComponents(nodeOptions.Directories)
	di.bootstrapLocationComponents(nodeOptions.Location, nodeOptions.Directories.Config)
	di.bootstrapNodeComponents(nodeOptions)
	di.bootstrapServiceOpenvpn(nodeOptions)

	if err := di.Node.Start(); err != nil {
		return err
	}

	return nil
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

	if di.ServiceManager != nil {
		if err := di.ServiceManager.Kill(); err != nil {
			errs = append(errs, err)
		}
	}
	if di.Node != nil {
		if err := di.Node.Kill(); err != nil {
			errs = append(errs, err)
		}
	}

	log.Flush()
	return nil
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options) {
	dialogFactory := func(consumerID, providerID identity.Identity, contact dto_discovery.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, di.SignerFactory(consumerID))
		return dialogEstablisher.EstablishDialog(providerID, contact)
	}

	promiseIssuerFactory := func(issuerID identity.Identity, dialog communication.Dialog) connection.PromiseIssuer {
		return noop.NewPromiseIssuer(issuerID, dialog, di.SignerFactory(issuerID))
	}

	di.StatsKeeper = stats.NewSessionStatsKeeper(time.Now)

	di.ConnectionManager = connection.NewManager(di.MysteriumClient, dialogFactory, promiseIssuerFactory, di.ConnectionCreators)
	di.ConnectionCreators = make(map[string]connection.ConnectionCreator)

	router := tequilapi.NewAPIRouter()
	tequilapi_endpoints.AddRouteForStop(router, utils.SoftKiller(di.Shutdown))
	tequilapi_endpoints.AddRoutesForIdentities(router, di.IdentityManager, di.MysteriumClient, di.SignerFactory)
	tequilapi_endpoints.AddRoutesForConnection(router, di.ConnectionManager, di.IPResolver, di.StatsKeeper)
	tequilapi_endpoints.AddRoutesForLocation(router, di.ConnectionManager, di.LocationDetector, di.LocationOriginal)
	tequilapi_endpoints.AddRoutesForProposals(router, di.MysteriumClient)
	identity_registry.AddIdentityRegistrationEndpoint(router, di.IdentityRegistration, di.IdentityRegistry)

	httpAPIServer := tequilapi.NewServer(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort, router)

	di.NodeOptions = nodeOptions
	di.Node = node.NewNode(di.ConnectionManager, httpAPIServer, di.LocationOriginal)
}

func (di *Dependencies) bootstrapServiceOpenvpn(nodeOptions node.Options) {
	di.ConnectionCreators["openvpn"] = openvpn.NewProcessBasedConnectionFactory(
		di.MysteriumClient,
		nodeOptions.Openvpn.BinaryPath,
		nodeOptions.Directories.Config,
		nodeOptions.Directories.Runtime,
		di.StatsKeeper,
		di.LocationOriginal,
		di.SignerFactory,
	)
}

// BootstrapServiceComponents initiates ServiceManager dependency
func (di *Dependencies) BootstrapServiceComponents(nodeOptions node.Options, serviceOptions service.Options) {
	identityHandler := identity_selector.NewHandler(
		di.IdentityManager,
		di.MysteriumClient,
		identity.NewIdentityCache(nodeOptions.Directories.Keystore, "remember.json"),
		di.SignerFactory,
	)
	identityLoader := identity_selector.NewLoader(identityHandler, serviceOptions.Identity, serviceOptions.Passphrase)

	discoveryService := discovery.NewService(di.IdentityRegistry, di.IdentityRegistration, di.MysteriumClient, di.SignerFactory)

	sessionStorage := session.NewStorageMemory()

	openvpnServiceManager := openvpn_service.NewManager(nodeOptions, serviceOptions, di.IPResolver, di.LocationResolver, sessionStorage)

	di.ServiceManager = service.NewManager(
		di.NetworkDefinition,
		identityLoader,
		di.SignerFactory,
		di.IdentityRegistry,
		openvpnServiceManager,
		func(proposal dto_discovery.ServiceProposal, configProvider session.ConfigProvider) communication.DialogHandler {
			sessionManagerFactory := newSessionManagerFactory(proposal, configProvider, sessionStorage)
			return session.NewDialogHandler(sessionManagerFactory)
		},
		discoveryService,
	)
}

func newSessionManagerFactory(
	proposal dto_discovery.ServiceProposal,
	configProvider session.ConfigProvider,
	sessionStorage *session.StorageMemory,
) session.ManagerFactory {
	return func(dialog communication.Dialog) session.Manager {
		promiseProcessor := noop.NewPromiseProcessor(dialog)
		return session.NewManager(
			proposal,
			session.GenerateUUID,
			configProvider,
			sessionStorage.Add,
			promiseProcessor,
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

	log.Info("Using Eth endpoint: ", network.EtherClientRPC)
	if di.EtherClient, err = blockchain.NewClient(network.EtherClientRPC); err != nil {
		return err
	}

	log.Info("Using Eth contract at address: ", network.PaymentsContractAddress.String())
	if di.IdentityRegistry, err = identity_registry.NewIdentityRegistryContract(di.EtherClient, network.PaymentsContractAddress); err != nil {
		return err
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
		di.LocationResolver = location.NewResolverFake(options.Country)
	default:
		di.LocationResolver = location.NewResolver(filepath.Join(configDirectory, options.Database))
	}

	di.LocationDetector = location.NewDetector(di.IPResolver, di.LocationResolver)
	di.LocationOriginal = location.NewLocationCache(di.LocationDetector)
}
