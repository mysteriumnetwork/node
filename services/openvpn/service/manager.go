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
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/traversal"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
)

const logPrefix = "[service-openvpn] "

// ServerConfigFactory callback generates session config for remote client
type ServerConfigFactory func(secPrimitives *tls.Primitives, port int) *openvpn_service.ServerConfig

// ServerFactory initiates Openvpn server instance during runtime
type ServerFactory func(*openvpn_service.ServerConfig) openvpn.Process

// ProposalFactory prepares service proposal during runtime
type ProposalFactory func(currentLocation market.Location) market.ServiceProposal

// SessionConfigNegotiatorFactory initiates ConfigProvider instance during runtime
type SessionConfigNegotiatorFactory func(secPrimitives *tls.Primitives, outboundIP, publicIP string, port int) session.ConfigNegotiator

// NATPinger defined Pinger interface for Provider
type NATPinger interface {
	BindPort(port int)
	WaitForHole() error
	Stop()
}

// NATEventGetter allows us to fetch the last known NAT event
type NATEventGetter interface {
	LastEvent() traversal.Event
}

type PortPool interface {
	Acquire() port.Port
}

// Manager represents entrypoint for Openvpn service with top level components
type Manager struct {
	natService     nat.NATService
	mapPort        func(int) (releasePortMapping func())
	portPool       PortPool
	port           port.Port
	releasePorts   func()
	natPinger      NATPinger
	natEventGetter NATEventGetter

	sessionConfigNegotiatorFactory SessionConfigNegotiatorFactory
	consumerConfig                 openvpn_service.ConsumerConfig

	vpnServerConfigFactory   ServerConfigFactory
	vpnServiceConfigProvider session.ConfigNegotiator
	vpnServerFactory         ServerFactory
	vpnServer                openvpn.Process

	publicIP        string
	outboundIP      string
	currentLocation string
	serviceOptions  Options
}

// Serve starts service - does block
func (m *Manager) Serve(providerID identity.Identity) (err error) {
	err = m.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIP:      m.outboundIP,
	})
	if err != nil {
		return errors.Wrap(err, "failed to add NAT forwarding rule")
	}

	m.port = m.portPool.Acquire()
	m.releasePorts = m.mapPort(m.port.Num())

	primitives, err := primitiveFactory(m.currentLocation, providerID.Address)
	if err != nil {
		return
	}

	m.vpnServiceConfigProvider = m.sessionConfigNegotiatorFactory(primitives, m.outboundIP, m.publicIP, m.port.Num())

	vpnServerConfig := m.vpnServerConfigFactory(primitives, m.port.Num())
	m.vpnServer = m.vpnServerFactory(vpnServerConfig)

	// block until NATPinger punches the hole in NAT for first incoming connect or continues if service not behind NAT
	m.natPinger.BindPort(m.serviceOptions.Port)

	log.Info(logPrefix, "starting openvpn server on port: ", m.port.Num())
	if err := firewall.AddInboundRule(m.serviceOptions.Protocol, m.port.Num()); err != nil {
		return errors.Wrap(err, "failed to add firewall rule")
	}

	if err = m.vpnServer.Start(); err != nil {
		return
	}
	log.Info(logPrefix, "openvpn server waiting")

	return m.vpnServer.Wait()
}

// Stop stops service
func (m *Manager) Stop() (err error) {
	if m.releasePorts != nil {
		m.releasePorts()
	}

	if err := firewall.RemoveInboundRule(m.serviceOptions.Protocol, m.port.Num()); err != nil {
		log.Error(logPrefix, "Failed to delete firewall rule for OpenVPN", err)
	}

	if m.vpnServer != nil {
		m.vpnServer.Stop()
	}

	m.port.Release()

	return nil
}

// ProvideConfig takes session creation config from end consumer and provides the service configuration to the end consumer
func (m *Manager) ProvideConfig(config json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	if m.vpnServiceConfigProvider == nil {
		log.Info(logPrefix, "Config provider not initialized")
		return nil, nil, errors.New("Config provider not initialized")
	}

	if config != nil && len(config) > 0 { // Older clients do not send any config, but we should keep back compatibility and not fail in this case.
		var c openvpn_service.ConsumerConfig
		err := json.Unmarshal(config, &c)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing consumer config failed")
		}
		m.consumerConfig = c
	}

	return m.vpnServiceConfigProvider.ProvideConfig(config)
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
