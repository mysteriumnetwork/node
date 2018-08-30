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

package server

import (
	"sync"

	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/blockchain"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/identity/registry"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/metadata"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/openvpn/tls"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/mysterium/node/tequilapi"
)

// Command represent entrypoint for Mysterium server with top level components
type Command struct {
	networkDefinition metadata.NetworkDefinition
	keystore          *keystore.KeyStore
	identityLoader    func() (identity.Identity, error)
	createSigner      identity.SignerFactory
	ipResolver        ip.Resolver
	mysteriumClient   server.Client
	natService        nat.NATService
	locationResolver  location.Resolver

	dialogWaiterFactory func(identity identity.Identity, identityRegistry registry.IdentityRegistry) communication.DialogWaiter
	dialogWaiter        communication.DialogWaiter

	sessionManagerFactory func(primitives *tls.Primitives, serverIP string) session.Manager

	vpnServerFactory func(sessionManager session.Manager, primitives *tls.Primitives, openvpnStateCallback state.Callback) openvpn.Process

	vpnServer                   openvpn.Process
	checkOpenvpn                func() error
	checkDirectories            func() error
	openvpnServiceAddress       func(string, string) string
	protocol                    string
	proposalAnnouncementStopped *sync.WaitGroup
	httpAPIServer               tequilapi.APIServer
	router                      *httprouter.Router
}

// Start starts server - does not block
func (cmd *Command) Start() (err error) {
	log.Infof("Starting Mysterium Server (%s)", metadata.VersionAsString())

	err = cmd.checkDirectories()
	if err != nil {
		return err
	}

	err = cmd.checkOpenvpn()
	if err != nil {
		return err
	}

	providerID, err := cmd.identityLoader()
	if err != nil {
		return err
	}

	ethClient, err := blockchain.NewClient(cmd.networkDefinition.EtherClientRPC)
	if err != nil {
		return err
	}

	identityRegistry, err := registry.NewIdentityRegistry(ethClient, cmd.networkDefinition.PaymentsContractAddress)
	if err != nil {
		return err
	}

	cmd.dialogWaiter = cmd.dialogWaiterFactory(providerID, identityRegistry)
	providerContact, err := cmd.dialogWaiter.Start()
	if err != nil {
		return err
	}

	publicIP, err := cmd.ipResolver.GetPublicIP()
	if err != nil {
		return err
	}

	// if for some reason we will need truly external IP, use GetPublicIP()
	outboundIP, err := cmd.ipResolver.GetOutboundIP()
	if err != nil {
		return err
	}

	cmd.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIP:      outboundIP,
	})

	err = cmd.natService.Start()
	if err != nil {
		log.Warn("received nat service error: ", err, " trying to proceed.")
	}

	currentCountry, err := cmd.locationResolver.ResolveCountry(publicIP)
	if err != nil {
		return err
	}
	log.Info("Country detected: ", currentCountry)
	serviceLocation := dto_discovery.Location{Country: currentCountry}

	primitives, err := tls.NewTLSPrimitives(serviceLocation, providerID)
	if err != nil {
		return err
	}

	registrationDataProvider := registry.NewRegistrationDataProvider(cmd.keystore)

	discoveryService := discovery.NewService(identityRegistry, providerID, registrationDataProvider, cmd.mysteriumClient, cmd.createSigner)
	stopDiscovery, err := discoveryService.Start(cmd.proposalAnnouncementStopped)
	if err != nil {
		return err
	}

	sessionManager := cmd.sessionManagerFactory(primitives, cmd.openvpnServiceAddress(outboundIP, publicIP))

	proposal := discoveryService.GenertateServiceProposalWithLocation(providerID, providerContact, serviceLocation, cmd.protocol)
	dialogHandler := session.NewDialogHandler(proposal.ID, sessionManager)
	if err := cmd.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return err
	}

	vpnStateCallback := func(state openvpn.State) {
		switch state {
		case openvpn.ProcessStarted:
			log.Info("Openvpn service booting up")
		case openvpn.ConnectedState:
			log.Info("Openvpn service started successfully")
		case openvpn.ProcessExited:
			log.Info("Openvpn service exited")
			stopDiscovery()
		}
	}
	cmd.vpnServer = cmd.vpnServerFactory(sessionManager, primitives, vpnStateCallback)
	if err := cmd.vpnServer.Start(); err != nil {
		return err
	}

	err = cmd.registerTequilAPI(identityRegistry, providerID, registrationDataProvider)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *Command) registerTequilAPI(statusProvider registry.IdentityRegistry, providerID identity.Identity, registrationDataProvider registry.RegistrationDataProvider) error {
	registry.AddRegistrationEndpoint(cmd.router, registrationDataProvider, statusProvider, &providerID)

	err := cmd.httpAPIServer.StartServing()
	if err != nil {
		return err
	}

	address, err := cmd.httpAPIServer.Address()
	if err != nil {
		return err
	}

	port, err := cmd.httpAPIServer.Port()
	if err != nil {
		return err
	}

	log.Infof("Api started on: %s:%d", address, port)
	return nil
}

// Wait blocks until server is stopped
func (cmd *Command) Wait() error {
	log.Info("Waiting for proposal announcements to finish")
	cmd.proposalAnnouncementStopped.Wait()
	log.Info("Waiting for vpn service to finish")
	return cmd.vpnServer.Wait()
}

// Kill stops server
func (cmd *Command) Kill() error {
	cmd.natService.Stop()
	cmd.vpnServer.Stop()

	err := cmd.dialogWaiter.Stop()
	if err != nil {
		return err
	}

	return err
}
