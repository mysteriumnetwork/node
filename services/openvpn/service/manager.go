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
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-openvpn] "

// ServerFactory initiates Openvpn server instance during runtime
type ServerFactory func(primitives *tls.Primitives) openvpn.Process

// ProposalFactory prepares service proposal during runtime
type ProposalFactory func(currentLocation dto_discovery.Location) dto_discovery.ServiceProposal

// SessionManagerFactory initiates session manager instance during runtime
type SessionManagerFactory func(primitives *tls.Primitives, outboundIP, publicIP string) session.Manager

// Manager represents entrypoint for Openvpn service with top level components
type Manager struct {
	ipResolver            ip.Resolver
	natService            nat.NATService
	locationResolver      location.Resolver
	proposalFactory       ProposalFactory
	sessionManagerFactory SessionManagerFactory

	vpnServerFactory ServerFactory
	vpnServer        openvpn.Process
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (
	proposal dto_discovery.ServiceProposal,
	sessionManager session.Manager,
	err error,
) {
	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return
	}

	// if for some reason we will need truly external IP, use GetPublicIP()
	outboundIP, err := manager.ipResolver.GetOutboundIP()
	if err != nil {
		return
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
		log.Warn(logPrefix, "Failed to detect service country. ", err)
		err = service.ErrorLocation
		return
	}
	currentLocation := dto_discovery.Location{Country: currentCountry}
	log.Info(logPrefix, "Country detected: ", currentCountry)

	caSubject := pkix.Name{
		Country:            []string{currentCountry},
		Organization:       []string{"Mysterium Network"},
		OrganizationalUnit: []string{"Mysterium Team"},
	}
	serverCertSubject := pkix.Name{
		Country:            []string{currentCountry},
		Organization:       []string{"Mysterium node operator company"},
		OrganizationalUnit: []string{"Node operator team"},
		CommonName:         providerID.Address,
	}

	primitives, err := tls.NewTLSPrimitives(caSubject, serverCertSubject)
	if err != nil {
		return
	}

	manager.vpnServer = manager.vpnServerFactory(primitives)
	if err = manager.vpnServer.Start(); err != nil {
		return
	}

	proposal = manager.proposalFactory(currentLocation)
	sessionManager = manager.sessionManagerFactory(primitives, outboundIP, publicIP)
	return
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	return manager.vpnServer.Wait()
}

// Stop stops service
func (manager *Manager) Stop() error {
	if manager.natService != nil {
		manager.natService.Stop()
	}

	if manager.vpnServer != nil {
		manager.vpnServer.Stop()
	}

	return nil
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
