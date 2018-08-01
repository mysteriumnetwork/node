/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package client

import (
	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/blockchain"
	"github.com/mysterium/node/blockchain/registry"
	"github.com/mysterium/node/client/connection"
	node_cmd "github.com/mysterium/node/cmd"
	"github.com/mysterium/node/communication"
	nats_dialog "github.com/mysterium/node/communication/nats/dialog"
	nats_discovery "github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/metadata"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi"
	tequilapi_endpoints "github.com/mysterium/node/tequilapi/endpoints"
	"path/filepath"
	"time"
)

// NewCommand function creates new client command by given options
func NewCommand(options CommandOptions) *Command {
	networkDefinition := node_cmd.GetNetworkDefinition(options.NetworkOptions)
	mysteriumClient := server.NewClient(networkDefinition.DiscoveryAPIAddress)

	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	keystoreDirectory := filepath.Join(options.Directories.Data, "keystore")
	keystoreInstance := keystore.NewKeyStore(keystoreDirectory, keystore.StandardScryptN, keystore.StandardScryptP)

	identityManager := identity.NewIdentityManager(keystoreInstance)

	dialogFactory := func(consumerID, providerID identity.Identity, contact dto.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, identity.NewSigner(keystoreInstance, consumerID))
		return dialogEstablisher.EstablishDialog(providerID, contact)
	}

	signerFactory := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(keystoreInstance, id)
	}

	statsKeeper := bytescount.NewSessionStatsKeeper(time.Now)

	ipResolver := ip.NewResolver(options.IpifyUrl)

	locationDetector := location.NewDetector(
		ipResolver,
		filepath.Join(options.Directories.Config, options.LocationDatabase),
	)

	originalLocationCache := location.NewLocationCache(locationDetector)

	vpnClientFactory := connection.ConfigureVpnClientFactory(
		mysteriumClient,
		options.OpenvpnBinary,
		options.Directories.Config,
		options.Directories.Runtime,
		signerFactory,
		statsKeeper,
		originalLocationCache,
	)
	connectionManager := connection.NewManager(mysteriumClient, dialogFactory, vpnClientFactory, statsKeeper)

	router := tequilapi.NewAPIRouter()

	httpAPIServer := tequilapi.NewServer(options.TequilapiAddress, options.TequilapiPort, router)

	tequilapi_endpoints.AddRoutesForIdentities(router, identityManager, mysteriumClient, signerFactory)
	tequilapi_endpoints.AddRoutesForConnection(router, connectionManager, ipResolver, statsKeeper)
	tequilapi_endpoints.AddRoutesForLocation(router, connectionManager, locationDetector, originalLocationCache)
	tequilapi_endpoints.AddRoutesForProposals(router, mysteriumClient)

	return &Command{
		options:               options,
		network:               networkDefinition,
		keystore:              keystoreInstance,
		router:                router,
		connectionManager:     connectionManager,
		httpAPIServer:         httpAPIServer,
		originalLocationCache: originalLocationCache,
	}
}

//Command represent entrypoint for Mysterium client with top level components
type Command struct {
	options               CommandOptions
	network               metadata.NetworkDefinition
	keystore              *keystore.KeyStore
	connectionManager     connection.Manager
	router                *httprouter.Router
	httpAPIServer         tequilapi.APIServer
	originalLocationCache location.Cache
}

// Start starts Tequilapi service, fetches location
func (cmd *Command) Start() error {
	log.Infof("Starting Mysterium Client (%s)", metadata.VersionAsString())

	err := openvpn.CheckOpenvpnBinary(cmd.options.OpenvpnBinary)
	if err != nil {
		return err
	}

	err = cmd.options.Directories.Check()
	if err != nil {
		return err
	}

	originalLocation, err := cmd.originalLocationCache.RefreshAndGet()
	if err != nil {
		log.Warn("Failed to detect original country: ", err)
	} else {
		log.Info("Original country detected: ", originalLocation.Country)
	}

	tequilapi_endpoints.AddRouteForStop(cmd.router, node_cmd.SoftKiller(cmd.Kill))

	log.Info("Using Eth endpoint: ", cmd.network.EtherClientRPC)
	client, err := blockchain.NewClient(cmd.network.EtherClientRPC)
	if err != nil {
		return err
	}

	log.Info("Using Contract at address:", cmd.network.PaymentsContractAddress.String())
	statusProvider, err := registry.NewIdentityRegistry(client, cmd.network.PaymentsContractAddress)
	if err != nil {
		return err
	}

	registry.AddRegistrationEndpoint(cmd.router, registry.NewRegistrationDataProvider(cmd.keystore), statusProvider)

	err = cmd.httpAPIServer.StartServing()
	if err != nil {
		return err
	}

	port, err := cmd.httpAPIServer.Port()
	if err != nil {
		return err
	}

	log.Infof("Api started on: %d", port)

	return nil
}

// Wait blocks until tequilapi service is stopped
func (cmd *Command) Wait() error {
	return cmd.httpAPIServer.Wait()
}

// Kill stops tequilapi service
func (cmd *Command) Kill() error {

	err := cmd.connectionManager.Disconnect()
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			log.Info("No active connection - proceeding")
		default:
			return err
		}
	} else {
		log.Info("Connection closed")
	}

	cmd.httpAPIServer.Stop()
	log.Info("Api stopped")

	return nil
}
