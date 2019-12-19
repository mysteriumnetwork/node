/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/firewall"
	nats_lib "github.com/nats-io/go-nats"
	"github.com/rs/zerolog/log"
)

// Broker Constants
const (
	BrokerPort          = 4222
	BrokerMaxReconnect  = -1
	BrokerReconnectWait = 1 * time.Second
	BrokerTimeout       = 5 * time.Second
)

// ParseServerURI validates given NATS server address
func ParseServerURI(uri string) (*url.URL, error) {
	// Add scheme first otherwise url.Parse() fails.
	if !strings.HasPrefix(uri, "nats:") {
		uri = fmt.Sprintf("nats://%s", uri)
	}

	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if url.Port() == "" {
		url.Host = fmt.Sprintf("%s:%d", url.Host, BrokerPort)
	}

	return url, nil
}

// NewConnection create new ConnectionWrap to given servers
func NewConnection(serverURIs ...string) (*ConnectionWrap, error) {
	connection := &ConnectionWrap{
		servers: make([]string, len(serverURIs)),
	}

	for i, server := range serverURIs {
		natsURL, err := ParseServerURI(server)
		if err != nil {
			return nil, err
		}
		connection.servers[i] = natsURL.String()
	}

	return connection, nil
}

// NewConnection creates connection instances and connects instantly
func OpenConnection(serverURIs ...string) (*ConnectionWrap, error) {
	connection, err := NewConnection(serverURIs...)
	if err != nil {
		return connection, err
	}

	return connection, connection.Open()
}

// ConnectionWrap defines wrapped connection to NATS server(s)
type ConnectionWrap struct {
	*nats_lib.Conn
	servers     []string
	removeRules func()
}

// Open starts the connection
func (c *ConnectionWrap) Open() (err error) {
	options := nats_lib.GetDefaultOptions()
	options.Servers = c.servers
	options.MaxReconnect = BrokerMaxReconnect
	options.ReconnectWait = BrokerReconnectWait
	options.Timeout = BrokerTimeout
	options.PingInterval = 10 * time.Second
	options.DisconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("Disconnected") }
	options.ReconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("Reconnected") }

	c.removeRules, err = firewall.AllowURLAccess(c.servers...)
	if err != nil {
		return err
	}

	c.Conn, err = options.Connect()
	return err
}

// Close destructs the connection
func (c *ConnectionWrap) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}

	if c.removeRules != nil {
		c.removeRules()
	}
	c.removeRules = func() {}
}

// Check checks the connection
func (c *ConnectionWrap) Check() error {
	// Flush sends ping request and tries to send all cached data.
	// It return an error if something wrong happened. All other requests
	// will be added to queue to be sent after reconnecting.
	return c.Conn.Flush()
}

// Servers returns list of currently connected servers
func (c *ConnectionWrap) Servers() []string {
	return c.servers
}
