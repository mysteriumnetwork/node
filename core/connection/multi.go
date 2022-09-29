/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/identity"
)

type multiConnectionManager struct {
	mu  sync.RWMutex
	cms map[int]Manager

	newConnectionManager func() Manager
}

// NewMultiConnectionManager create a wrapper around connection manager to support multiple connections.
func NewMultiConnectionManager(newConnectionManager func() Manager) *multiConnectionManager {
	return &multiConnectionManager{
		cms: make(map[int]Manager),

		newConnectionManager: newConnectionManager,
	}
}

// Connect creates new connection from given consumer to provider, reports error if connection already exists.
func (mcm *multiConnectionManager) Connect(consumerID identity.Identity, hermesID common.Address, proposalLookup ProposalLookup, params ConnectParams) error {
	mcm.mu.Lock()

	m, ok := mcm.cms[params.ProxyPort]
	if !ok {
		m = mcm.newConnectionManager()
		mcm.cms[params.ProxyPort] = m
	}
	mcm.mu.Unlock()

	return m.Connect(consumerID, hermesID, proposalLookup, params)
}

// Status queries current status of connection.
func (mcm *multiConnectionManager) Status(id int) connectionstate.Status {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	if m, ok := mcm.cms[id]; ok {
		return m.Status()
	}

	return connectionstate.Status{
		State: connectionstate.NotConnected,
	}
}

// Stats provides connection statistics information.
func (mcm *multiConnectionManager) Stats(id int) connectionstate.Statistics {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	if m, ok := mcm.cms[id]; ok {
		return m.Stats()
	}

	return connectionstate.Statistics{}
}

// Disconnect closes established connection, reports error if no connection.
func (mcm *multiConnectionManager) Disconnect(id int) error {
	mcm.mu.RLock()
	m, ok := mcm.cms[id]
	mcm.mu.RUnlock()

	if ok {
		err := m.Disconnect()
		return err
	}

	if id < 0 {
		mcm.mu.RLock()
		defer mcm.mu.RUnlock()

		for _, m := range mcm.cms {
			if err := m.Disconnect(); err != nil {
				log.Error().Err(err).Msg("Failed to disconnect active connection")
			}
		}
	}

	return nil
}

// CheckChannel checks if current session channel is alive, returns error on failed keep-alive ping.
func (mcm *multiConnectionManager) CheckChannel(context.Context) error { return nil }

// Reconnect reconnects current session.
func (mcm *multiConnectionManager) Reconnect(id int) {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	if m, ok := mcm.cms[id]; ok {
		m.Reconnect()
	}
}
