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

package node

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/julienschmidt/httprouter"

	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/openvpn"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/utils"
)

// NewNode function creates new Mysterium node by given options
func NewNode(
	options Options,
	keystoreInstance *keystore.KeyStore,
	identityManager identity.Manager,
	signerFactory identity.SignerFactory,
	identityRegistry identity_registry.IdentityRegistry,
	mysteriumClient server.Client,
	ipResolver ip.Resolver,
	locationResolver location.Resolver,
) *Node {
	logconfig.Bootstrap()
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	dialogFactory := func(consumerID, providerID identity.Identity, contact dto.Contact) (communication.Dialog, error) {
		dialogEstablisher := nats_dialog.NewDialogEstablisher(consumerID, signerFactory(consumerID))
		return dialogEstablisher.EstablishDialog(providerID, contact)
	}

	statsKeeper := stats.NewSessionStatsKeeper(time.Now)

	locationDetector := location.NewDetector(ipResolver, locationResolver)
	originalLocationCache := location.NewLocationCache(locationDetector)

	vpnClientFactory := connection.ConfigureVpnClientFactory(
		mysteriumClient,
		options.Openvpn.Binary,
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
	identity_registry.AddRegistrationEndpoint(router, identity_registry.NewRegistrationDataProvider(keystoreInstance), identityRegistry)

	return &Node{
		options:               options,
		router:                router,
		connectionManager:     connectionManager,
		httpAPIServer:         httpAPIServer,
		originalLocationCache: originalLocationCache,
	}
}

// Node represent entrypoint for Mysterium node with top level components
type Node struct {
	options               Options
	connectionManager     connection.Manager
	router                *httprouter.Router
	httpAPIServer         tequilapi.APIServer
	originalLocationCache location.Cache
}

// Start starts Mysterium node (Tequilapi service, fetches location)
func (node *Node) Start() error {
	log.Infof("Starting Mysterium Client (%s)", metadata.VersionAsString())

	err := openvpn.CheckOpenvpnBinary(node.options.Openvpn.Binary)
	if err != nil {
		return err
	}

	originalLocation, err := node.originalLocationCache.RefreshAndGet()
	if err != nil {
		log.Warn("Failed to detect original country: ", err)
	} else {
		log.Info("Original country detected: ", originalLocation.Country)
	}

	tequilapi_endpoints.AddRouteForStop(node.router, utils.SoftKiller(node.Kill))

	err = node.httpAPIServer.StartServing()
	if err != nil {
		return err
	}

	port, err := node.httpAPIServer.Port()
	if err != nil {
		return err
	}

	log.Infof("Api started on: %d", port)

	return nil
}

// Wait blocks until Mysterium node is stopped
func (node *Node) Wait() error {
	return node.httpAPIServer.Wait()
}

// Kill stops Mysterium node
func (node *Node) Kill() error {
	err := node.connectionManager.Disconnect()
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

	node.httpAPIServer.Stop()
	log.Info("Api stopped")

	log.Flush()

	return nil
}
