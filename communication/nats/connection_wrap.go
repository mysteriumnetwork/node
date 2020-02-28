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
	"sync"
	"time"

	nats_lib "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// DefaultBrokerPort broker port.
	DefaultBrokerPort = 4222
)

// ParseServerURI validates given NATS server address.
func ParseServerURI(serverURI string) (*url.URL, error) {
	// Add scheme first otherwise serverURL.Parse() fails.
	if !strings.HasPrefix(serverURI, "nats:") {
		serverURI = fmt.Sprintf("nats://%s", serverURI)
	}

	serverURL, err := url.Parse(serverURI)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to parse NATS server URI "%s"`, serverURI)
	}
	if serverURL.Port() == "" {
		serverURL.Host = fmt.Sprintf("%s:%d", serverURL.Host, DefaultBrokerPort)
	}

	return serverURL, nil
}

func newConnection(serverURIs ...string) (*ConnectionWrap, error) {
	connection := &ConnectionWrap{
		servers: make([]string, len(serverURIs)),
		onClose: func() {},
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

// ConnectionWrap defines wrapped connection to NATS server(s).
type ConnectionWrap struct {
	conn     *nats_lib.Conn
	connLock sync.RWMutex

	servers []string
	onClose func()
}

func (c *ConnectionWrap) connectOptions() nats_lib.Options {
	options := nats_lib.GetDefaultOptions()
	options.Servers = c.servers
	options.MaxReconnect = -1
	options.ReconnectWait = 1 * time.Second
	options.Timeout = 5 * time.Second
	options.PingInterval = 10 * time.Second
	options.ClosedCB = func(conn *nats_lib.Conn) { log.Warn().Msg("NATS: connection closed") }
	options.DisconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("NATS: disconnected") }
	options.ReconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("NATS: reconnected") }
	return options
}

// Open starts the connection: left for test compatibility.
// Deprecated: Use nats.BrokerConnector#Connect() instead.
func (c *ConnectionWrap) Open() (err error) {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	c.conn, err = c.connectOptions().Connect()
	if err != nil {
		return errors.Wrapf(err, `failed to connect to NATS servers "%v"`, c.servers)
	}

	return nil
}

// Reopen restarts the connection.
func (c *ConnectionWrap) Reopen() (err error) {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	c.conn, err = c.connectOptions().Connect()
	if err != nil {
		return errors.Wrapf(err, `failed to reconnect to NATS servers "%v"`, c.servers)
	}

	return nil
}

// Close destructs the connection.
func (c *ConnectionWrap) Close() {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
	c.onClose()
}

// Check checks the connection.
func (c *ConnectionWrap) Check() error {
	// Flush sends ping request and tries to send all cached data.
	// It return an error if something wrong happened. All other requests
	// will be added to queue to be sent after reconnecting.
	return c.conn.FlushTimeout(3 * time.Second)
}

// Servers returns list of currently connected servers.
func (c *ConnectionWrap) Servers() []string {
	return c.servers
}

// Publish publishes payload  to the given subject.
func (c *ConnectionWrap) Publish(subject string, payload []byte) error {
	c.connLock.RLock()
	defer c.connLock.RUnlock()

	return c.conn.Publish(subject, payload)
}

// Subscribe will express interest in the given subject.
func (c *ConnectionWrap) Subscribe(subject string, handler nats_lib.MsgHandler) (*nats_lib.Subscription, error) {
	c.connLock.RLock()
	defer c.connLock.RUnlock()

	return c.conn.Subscribe(subject, handler)
}

// Request will send a request payload and deliver the response message.
func (c *ConnectionWrap) Request(subject string, payload []byte, timeout time.Duration) (*nats_lib.Msg, error) {
	c.connLock.RLock()
	defer c.connLock.RUnlock()

	return c.conn.Request(subject, payload, timeout)
}
