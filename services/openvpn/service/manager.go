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
	"encoding/json"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
)

const logPrefix = "[service-openvpn] "

// ServerConfigFactory callback generates session config for remote client
type ServerConfigFactory func(*tls.Primitives) *openvpn_service.ServerConfig

// ServerFactory initiates Openvpn server instance during runtime
type ServerFactory func(*openvpn_service.ServerConfig) openvpn.Process

// ProposalFactory prepares service proposal during runtime
type ProposalFactory func(currentLocation market.Location) market.ServiceProposal

// SessionConfigNegotiatorFactory initiates ConfigProvider instance during runtime
type SessionConfigNegotiatorFactory func(secPrimitives *tls.Primitives, outboundIP, publicIP string) session.ConfigNegotiator

// Manager represents entrypoint for Openvpn service with top level components
type Manager struct {
	natService nat.NATService

	sessionConfigNegotiatorFactory SessionConfigNegotiatorFactory

	vpnServerConfigFactory   ServerConfigFactory
	vpnServiceConfigProvider session.ConfigNegotiator
	vpnServerFactory         ServerFactory
	vpnServer                openvpn.Process

	publicIP        string
	outboundIP      string
	currentLocation string
}

// Serve starts service - does block
func (manager *Manager) Serve(providerID identity.Identity) (err error) {
	err = manager.natService.Enable()
	if err != nil {
		log.Warn(logPrefix, "received nat service error: ", err, " trying to proceed.")
	}

	err = manager.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIP:      manager.outboundIP,
	})
	if err != nil {
		return errors.Wrap(err, "failed to add NAT forwarding rule")
	}

	primitives, err := primitiveFactory(manager.currentLocation, providerID.Address)
	if err != nil {
		return
	}

	manager.vpnServiceConfigProvider = manager.sessionConfigNegotiatorFactory(primitives, manager.outboundIP, manager.publicIP)

	vpnServerConfig := manager.vpnServerConfigFactory(primitives)
	manager.vpnServer = manager.vpnServerFactory(vpnServerConfig)

	if err = manager.vpnServer.Start(); err != nil {
		return
	}
	return manager.vpnServer.Wait()
}

// Stop stops service
func (manager *Manager) Stop() (err error) {
	if manager.natService != nil {
		err = manager.natService.Disable()
	}

	if manager.vpnServer != nil {
		manager.vpnServer.Stop()
	}

	return err
}

// ProvideConfig provides the configuration to end consumer
func (manager *Manager) ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	if manager.vpnServiceConfigProvider == nil {
		log.Info(logPrefix, "Config provider not initialized")
		return nil, nil, errors.New("Config provider not initialized")
	}

	return manager.vpnServiceConfigProvider.ProvideConfig(publicKey)
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
