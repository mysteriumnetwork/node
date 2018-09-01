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
	"github.com/mysteriumnetwork/node/identity"
	identity_loading "github.com/mysteriumnetwork/node/identity/loading"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/server"
)

// Dependencies is DI container for top level components which is reusedin several places
type Dependencies struct {
	NodeOptions node.Options
	Node        *node.Node

	NetworkDefinition metadata.NetworkDefinition
	MysteriumClient   server.Client
	EtherClient       *ethclient.Client

	Keystore         *keystore.KeyStore
	IdentityManager  identity.Manager
	SignerFactory    identity.SignerFactory
	IdentityRegistry identity_registry.IdentityRegistry

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

	if err := di.bootstrapNetwork(nodeOptions.NetworkOptions); err != nil {
		return err
	}

	di.bootstrapIdentity(nodeOptions.Directories)
	di.bootstrapLocation(nodeOptions.Location, nodeOptions.Directories.Config)
	di.bootstrapNode(nodeOptions)

	return nil
}

func (di *Dependencies) bootstrapNode(nodeOptions node.Options) {
	di.NodeOptions = nodeOptions
	di.Node = node.NewNode(
		nodeOptions,
		di.Keystore,
		di.IdentityManager,
		di.SignerFactory,
		di.IdentityRegistry,
		di.MysteriumClient,
		di.IPResolver,
		di.LocationResolver,
	)
}

// BootstrapServiceManager initiates ServiceManager dependency
func (di *Dependencies) BootstrapServiceManager(nodeOptions node.Options, serviceOptions service.Options) {
	identityHandler := identity_loading.NewHandler(
		di.IdentityManager,
		di.MysteriumClient,
		identity.NewIdentityCache(nodeOptions.Directories.Keystore, "remember.json"),
		di.SignerFactory,
	)
	identityLoader := identity_loading.NewLoader(identityHandler, serviceOptions.Identity, serviceOptions.Passphrase)

	di.ServiceManager = service.NewManager(
		nodeOptions,
		serviceOptions,
		di.NetworkDefinition,
		identityLoader,
		di.SignerFactory,
		di.IdentityRegistry,
		di.MysteriumClient,
		di.IPResolver,
		di.LocationResolver,
	)
}

// function decides on etwork definition combined from testnet/localnet flags and possible overrides
func (di *Dependencies) bootstrapNetwork(options node.NetworkOptions) (err error) {
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
	if normalizedAddress.String() != metadata.DefaultNetwork.PaymentsContractAddress.String() {
		network.PaymentsContractAddress = common.HexToAddress(options.EtherPaymentsAddress)
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

	log.Info("Using Contract at address:", network.PaymentsContractAddress.String())
	if di.IdentityRegistry, err = identity_registry.NewIdentityRegistry(di.EtherClient, network.PaymentsContractAddress); err != nil {
		return err
	}

	return nil
}

func (di *Dependencies) bootstrapIdentity(directories node.DirectoryOptions) {
	di.Keystore = identity.NewKeystoreFilesystem(directories.Keystore)
	di.IdentityManager = identity.NewIdentityManager(di.Keystore)
	di.SignerFactory = func(id identity.Identity) identity.Signer {
		return identity.NewSigner(di.Keystore, id)
	}
}

func (di *Dependencies) bootstrapLocation(options node.LocationOptions, configDirectory string) {
	di.IPResolver = ip.NewResolver(options.IpifyUrl)

	switch {
	case options.Country != "":
		di.LocationResolver = location.NewResolverFake(options.Country)
	default:
		di.LocationResolver = location.NewResolver(filepath.Join(configDirectory, options.Database))
	}
}
