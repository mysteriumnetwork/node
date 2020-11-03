/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package broker

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/market"
	"github.com/rs/zerolog/log"
)

// NewManager creates new instance of broker service.
func NewManager(addr string) *Manager {
	return &Manager{
		listenAddr: addr,
		handler:    newP2PHandler(),
	}
}

// Manager represents entrypoint for broker service.
type Manager struct {
	listenAddr string
	handler    *handler
}

// ProvideConfig provides the session configuration.
func (m *Manager) ProvideConfig(sessionID string, sessionConfig json.RawMessage, _ *net.UDPConn) (*service.ConfigParams, error) {
	out, err := sessionConfig.MarshalJSON()
	log.Info().Err(err).Msgf("New broker service session: %s (%s)", sessionID, string(out))
	return &service.ConfigParams{SessionServiceConfig: "http://localhost:12345", SessionDestroyCallback: func() {}}, nil
}

// Serve starts service - does block
func (m *Manager) Serve(instance *service.Instance) error {
	http.HandleFunc("/msg/", m.handler.brokerMsgHandle)
	http.HandleFunc("/poll/init/", m.handler.brokerPollInitHandle)
	http.HandleFunc("/poll/ack/", m.handler.brokerPollAckHandle)

	log.Info().Msg("Broker service started successfully")
	return http.ListenAndServe(m.listenAddr, nil)
}

// Stop stops service
func (m *Manager) Stop() error {
	log.Info().Msg("broker service stopped")
	return nil
}

// GetProposal returns the proposal for broker service for given country.
func GetProposal(location locationstate.Location) market.ServiceProposal {
	return market.ServiceProposal{
		ServiceType: ServiceType,
		ServiceDefinition: ServiceDefinition{
			Location: market.Location{
				Continent: location.Continent,
				Country:   location.Country,
				City:      location.City,

				ASN:      location.ASN,
				ISP:      location.ISP,
				NodeType: location.NodeType,
			},
		},
	}
}
