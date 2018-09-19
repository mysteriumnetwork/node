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

	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/node/blockchain"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/server"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
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

	ServiceManager *service.Manager
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) error {
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

	return nil
}

func (di *Dependencies) bootstrapNodeComponents(nodeOptions node.Options) {
	di.NodeOptions = nodeOptions
	di.Node = node.NewNode(
		nodeOptions,
		di.IdentityManager,
		di.SignerFactory,
		di.IdentityRegistry,
		di.IdentityRegistration,
		di.MysteriumClient,
		di.IPResolver,
		di.LocationResolver,
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

	openvpnServiceManager := openvpn_service.NewManager(nodeOptions, serviceOptions, di.IPResolver, di.LocationResolver)

	di.ServiceManager = service.NewManager(
		di.NetworkDefinition,
		identityLoader,
		di.SignerFactory,
		di.IdentityRegistry,
		openvpnServiceManager,
		discoveryService,
	)
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
}
