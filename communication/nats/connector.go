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
	"net/url"

	"github.com/mysteriumnetwork/node/firewall"
	nats_lib "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// BrokerConnector establishes new connections to NATS servers and handles reconnects.
type BrokerConnector struct {
	// Dialer specifies the custom dialer for creating unencrypted TCP connections.
	// If Dialer is nil, then the connector dials using package net.
	Dialer nats_lib.CustomDialer
}

// NewBrokerConnector creates a new BrokerConnector.
func NewBrokerConnector() *BrokerConnector {
	return &BrokerConnector{}
}

// Connect establishes a new connection to the broker(s).
func (b *BrokerConnector) Connect(serverURLs ...*url.URL) (Connection, error) {
	log.Debug().Msgf("Connecting to NATS servers: %v", serverURLs)

	servers := make([]string, len(serverURLs))
	for i, serverURL := range serverURLs {
		servers[i] = serverURL.String()
	}

	removeFirewallRule, err := firewall.AllowURLAccess(servers...)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to allow NATS servers "%v" in firewall`, servers)
	}

	conn := newConnectionWith(b.Dialer, servers...)
	if err := conn.Open(); err != nil {
		return nil, errors.Wrapf(err, `failed to connect to NATS servers "%v"`, servers)
	}

	conn.onClose = removeFirewallRule

	return conn, nil
}
