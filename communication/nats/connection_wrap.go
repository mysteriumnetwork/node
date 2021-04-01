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
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	nats_lib "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/requests"
)

const (
	// DefaultBrokerScheme broker scheme.
	DefaultBrokerScheme = "nats"
	// DefaultBrokerPort broker port.
	DefaultBrokerPort = 4222
)

// ParseServerURL validates given NATS server address.
func ParseServerURL(serverURI string) (*url.URL, error) {
	// Add scheme first otherwise serverURL.Parse() fails.
	if !strings.HasPrefix(serverURI, DefaultBrokerScheme) {
		serverURI = fmt.Sprintf("%s://%s", DefaultBrokerScheme, serverURI)
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

// ParseServerURIs validates given list of NATS server addresses.
func ParseServerURIs(serverURIs []string) ([]*url.URL, error) {
	serverURLs := make([]*url.URL, len(serverURIs))

	for i, server := range serverURIs {
		natsURL, err := ParseServerURL(server)
		if err != nil {
			return nil, err
		}

		serverURLs[i] = natsURL
	}

	return serverURLs, nil
}

func newConnection(dialer requests.DialContext, serverURIs ...string) (*ConnectionWrap, error) {
	return &ConnectionWrap{
		servers: serverURIs,
		onClose: func() {},
		dialer:  dialer,
	}, nil
}

// ConnectionWrap defines wrapped connection to NATS server(s).
type ConnectionWrap struct {
	*nats_lib.Conn

	dialer requests.DialContext

	servers []string
	onClose func()
}

func (c *ConnectionWrap) connectOptions() nats_lib.Options {
	options := nats_lib.GetDefaultOptions()
	options.CustomDialer = &dialer{c.dialer}
	options.Servers = c.servers
	options.MaxReconnect = -1
	options.ReconnectWait = 1 * time.Second
	options.PingInterval = 10 * time.Second
	options.ClosedCB = func(conn *nats_lib.Conn) { log.Warn().Msg("NATS: connection closed") }
	options.DisconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("NATS: disconnected") }
	options.ReconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("NATS: reconnected") }

	return options
}

// Open starts the connection: left for test compatibility.
// Deprecated: Use nats.BrokerConnector#Connect() instead.
func (c *ConnectionWrap) Open() (err error) {
	c.Conn, err = c.connectOptions().Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to NATS servers %v: %w", c.connectOptions().Servers, err)
	}

	return nil
}

// Close destructs the connection.
func (c *ConnectionWrap) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.onClose()
}

// Servers returns list of currently connected servers.
func (c *ConnectionWrap) Servers() []string {
	return c.servers
}

type dialer struct {
	dialer requests.DialContext
}

func (d *dialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer(context.Background(), network, address)
}
