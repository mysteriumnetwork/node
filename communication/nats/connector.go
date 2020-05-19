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

	"github.com/mysteriumnetwork/node/firewall"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// BrokerConnector establishes new connections to NATS servers and handles reconnects.
type BrokerConnector struct {
}

// NewBrokerConnector creates a new BrokerConnector.
func NewBrokerConnector() *BrokerConnector {
	return &BrokerConnector{}
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

	conn.onClose = removeFirewallRule

	return conn, nil
}
