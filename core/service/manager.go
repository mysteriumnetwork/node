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

package service

import (
	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/blockchain"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/ip"
	"github.com/mysteriumnetwork/node/location"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/openvpn"
	"github.com/mysteriumnetwork/node/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/openvpn/tls"
	"github.com/mysteriumnetwork/node/server"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

// Manager represent entrypoint for Mysterium service with top level components
type Manager struct {
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

	vpnServer             openvpn.Process
	checkOpenvpn          func() error
	checkDirectories      func() error
	openvpnServiceAddress func(string, string) string
	protocol              string
	discoveryService      *discovery.Discovery
}

const logPrefix = "[manager] "

// Start starts service - does not block
func (manager *Manager) Start() (err error) {
	log.Infof(logPrefix, "Starting Mysterium Server (%s)", metadata.VersionAsString())

	err = manager.checkDirectories()
	if err != nil {
		return err
	}

	err = manager.checkOpenvpn()
	if err != nil {
		return err
	}

	providerID, err := manager.identityLoader()
	if err != nil {
		return err
	}

	ethClient, err := blockchain.NewClient(manager.networkDefinition.EtherClientRPC)
	if err != nil {
		return err
	}

	identityRegistry, err := registry.NewIdentityRegistry(ethClient, manager.networkDefinition.PaymentsContractAddress)
	if err != nil {
		return err
	}

	manager.dialogWaiter = manager.dialogWaiterFactory(providerID, identityRegistry)
	providerContact, err := manager.dialogWaiter.Start()
	if err != nil {
		return err
	}

	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return err
	}

	// if for some reason we will need truly external IP, use GetPublicIP()
	outboundIP, err := manager.ipResolver.GetOutboundIP()
	if err != nil {
		return err
	}

	manager.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIP:      outboundIP,
	})

	err = manager.natService.Start()
	if err != nil {
		log.Warn(logPrefix, "received nat service error: ", err, " trying to proceed.")
	}

	currentCountry, err := manager.locationResolver.ResolveCountry(publicIP)
	if err != nil {
		return err
	}
	log.Info(logPrefix, "Country detected: ", currentCountry)
	serviceLocation := dto_discovery.Location{Country: currentCountry}

	primitives, err := tls.NewTLSPrimitives(serviceLocation, providerID)
	if err != nil {
		return err
	}

	registrationDataProvider := registry.NewRegistrationDataProvider(manager.keystore)

	manager.discoveryService = discovery.NewService(identityRegistry, providerID, registrationDataProvider, manager.mysteriumClient, manager.createSigner)
	manager.discoveryService.Start()

	sessionManager := manager.sessionManagerFactory(primitives, manager.openvpnServiceAddress(outboundIP, publicIP))

	proposal := manager.discoveryService.GenertateServiceProposalWithLocation(providerID, providerContact, serviceLocation, manager.protocol)
	dialogHandler := session.NewDialogHandler(proposal.ID, sessionManager)
	if err := manager.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
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
		}
	}
	manager.vpnServer = manager.vpnServerFactory(sessionManager, primitives, vpnStateCallback)
	if err := manager.vpnServer.Start(); err != nil {
		return err
	}

	return nil
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	log.Info(logPrefix, "Waiting for discovery service to finish")
	manager.discoveryService.Wait()
	log.Info(logPrefix, "Waiting for vpn service to finish")
	return manager.vpnServer.Wait()
}

// Kill stops service
func (manager *Manager) Kill() error {
	if manager.discoveryService != nil {
		manager.discoveryService.Stop()
	}

	var err error
	if manager.dialogWaiter != nil {
		err = manager.dialogWaiter.Stop()
	}

	if manager.natService != nil {
		manager.natService.Stop()
	}

	if manager.vpnServer != nil {
		manager.vpnServer.Stop()
	}

	return err
}
