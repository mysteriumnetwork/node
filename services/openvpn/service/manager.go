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
	"crypto/x509/pkix"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/nat"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	openvpn_discovery "github.com/mysteriumnetwork/node/services/openvpn/discovery"
	"github.com/mysteriumnetwork/node/session"
)

// Manager represent entrypoint for Mysterium service with top level components
type Manager struct {
	identityLoader   identity_selector.Loader
	ipResolver       ip.Resolver
	natService       nat.NATService
	locationResolver location.Resolver

	dialogWaiterFactory func(identity identity.Identity) communication.DialogWaiter
	dialogWaiter        communication.DialogWaiter

	sessionManagerFactory func(primitives *tls.Primitives, outboundIP, publicIP string) session.Manager

	vpnServerFactory func(primitives *tls.Primitives) openvpn.Process
	vpnServer        openvpn.Process

	protocol  string
	discovery *discovery.Discovery
}

const logPrefix = "[service-manager] "

// Start starts service - does not block
func (manager *Manager) Start() (err error) {
	log.Infof(logPrefix, "Starting Mysterium Server (%s)", metadata.VersionAsString())

	providerID, err := manager.identityLoader()
	if err != nil {
		return err
	}

	manager.dialogWaiter = manager.dialogWaiterFactory(providerID)
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

	proposal := openvpn_discovery.NewServiceProposalWithLocation(providerID, providerContact, serviceLocation, manager.protocol)

	caSubject := pkix.Name{
		Country:            []string{serviceLocation.Country},
		Organization:       []string{"Mystermium.network"},
		OrganizationalUnit: []string{"Mysterium Team"},
	}
	serverCertSubject := pkix.Name{
		Country:            []string{serviceLocation.Country},
		Organization:       []string{"Mysterium node operator company"},
		OrganizationalUnit: []string{"Node operator team"},
		CommonName:         providerID.Address,
	}

	primitives, err := tls.NewTLSPrimitives(caSubject, serverCertSubject)
	if err != nil {
		return err
	}

	manager.discovery.Start(providerID, proposal)

	sessionManager := manager.sessionManagerFactory(primitives, outboundIP, publicIP)

	dialogHandler := session.NewDialogHandler(proposal.ID, sessionManager)
	if err := manager.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return err
	}

	manager.vpnServer = manager.vpnServerFactory(primitives)
	if err := manager.vpnServer.Start(); err != nil {
		return err
	}

	return nil
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	log.Info(logPrefix, "Waiting for discovery service to finish")
	manager.discovery.Wait()

	log.Info(logPrefix, "Waiting for vpn service to finish")
	return manager.vpnServer.Wait()
}

// Kill stops service
func (manager *Manager) Kill() error {
	if manager.discovery != nil {
		manager.discovery.Stop()
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

func vpnStateCallback(state openvpn.State) {
	switch state {
	case openvpn.ProcessStarted:
		log.Info(logPrefix, "Openvpn service booting up")
	case openvpn.ConnectedState:
		log.Info(logPrefix, "Openvpn service started successfully")
	case openvpn.ProcessExited:
		log.Info(logPrefix, "Openvpn service exited")
	}
}
