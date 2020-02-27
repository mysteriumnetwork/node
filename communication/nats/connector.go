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

package nats

import (
	"strings"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// BrokerConnector establishes new connections to NATS servers and handles reconnects.
type BrokerConnector struct {
	registry map[uuid.UUID]*ConnectionWrap
	mu       sync.Mutex
}

// NewBrokerConnector creates a new BrokerConnector.
func NewBrokerConnector() *BrokerConnector {
	return &BrokerConnector{
		registry: make(map[uuid.UUID]*ConnectionWrap),
	}
}

// Connect establishes a new connection to the broker(s).
func (b *BrokerConnector) Connect(serverURIs ...string) (Connection, error) {
	log.Debug().Msg("Connecting to NATS servers: " + strings.Join(serverURIs, ","))

	conn, err := newConnection(serverURIs...)
	if err != nil {
		return nil, err
	}

	removeFirewallRule, err := firewall.AllowURLAccess(conn.servers...)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to allow NATS servers "%v" in firewall`, conn.servers)
	}

	if err := conn.Open(); err != nil {
		return nil, errors.Wrapf(err, `failed to connect to NATS servers "%v"`, conn.servers)
	}
	id, err := uuid.NewV4()
	if err != nil {
		removeFirewallRule()
		return nil, errors.Wrap(err, "could not generate UUID for the new connection")
	}

	b.mu.Lock()
	b.registry[id] = conn
	b.mu.Unlock()
	conn.onClose = func() {
		b.mu.Lock()
		log.Info().Msgf("Removing broker connection from the registry: %v", id)
		delete(b.registry, id)
		removeFirewallRule()
		b.mu.Unlock()
	}

	return conn, nil
}

// ReconnectAll checks all established connections to trigger reconnects.
func (b *BrokerConnector) ReconnectAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	log.Info().Msgf("Attempting to reconnect to broker (%d connections)", len(b.registry))
	var wg sync.WaitGroup
	for k, v := range b.registry {
		wg.Add(1)
		go func(id uuid.UUID, conn *ConnectionWrap) {
			defer wg.Done()
			log.Info().Msgf("Re-establishing broker connection %v", id)
			err := conn.Conn.Flush()
			log.Info().Msgf("Re-establishing broker connection %v DONE (check result=%v)", id, err)
		}(k, v)
	}
	wg.Wait()
	log.Info().Msg("All broker connections re-established")
}
