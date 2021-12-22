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
	"sync"
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
	// Timeout for NATS connect operations
	Timeout = 10 * time.Second
	// PingInterval between keepalive probes
	PingInterval = 10 * time.Second
	// ReconnectWait is a delay between reconnection attempts for single connection
	// and attempts between attempts to spawn new connection in place of failed one.
	ReconnectWait = 1 * time.Second
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
		closeCh: make(chan struct{}),
	}, nil
}

// ConnectionWrap defines wrapped connection to NATS server(s).
type ConnectionWrap struct {
	conn    *nats_lib.Conn
	connMux sync.RWMutex

	dialer requests.DialContext

	servers   []string
	onClose   func()
	closeOnce sync.Once
	closeCh   chan struct{}
}

func (c *ConnectionWrap) connectOptions() nats_lib.Options {
	options := nats_lib.GetDefaultOptions()
	options.Servers = c.servers
	options.MaxReconnect = -1
	options.Timeout = Timeout
	options.RetryOnFailedConnect = true
	options.NoCallbacksAfterClientClose = true
	options.ReconnectWait = ReconnectWait
	options.PingInterval = PingInterval
	options.ClosedCB = func(conn *nats_lib.Conn) {
		log.Warn().Msg("NATS: connection closed. Scheduling resume...")
		go c.resurrect()
	}
	options.DisconnectedErrCB = func(nc *nats_lib.Conn, err error) { log.Warn().Err(err).Msg("NATS: disconnected") }
	options.ReconnectedCB = func(nc *nats_lib.Conn) { log.Warn().Msg("NATS: reconnected") }

	if c.dialer != nil {
		options.CustomDialer = &dialer{c.dialer}
	}

	return options
}

func (c *ConnectionWrap) resurrect() {
	for {
		select {
		case <-c.closeCh:
			return
		case <-time.After(ReconnectWait):
			err := c.open()
			if err == nil {
				return
			}
		}
	}
}

// open starts the connection: left for test compatibility.
// Deprecated: Use nats.BrokerConnector#Connect() instead.
func (c *ConnectionWrap) open() (err error) {
	newConn, err := c.connectOptions().Connect()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to connect to NATS servers %v, will reconnect again", c.connectOptions().Servers)
		return err
	}

	c.connMux.Lock()
	defer c.connMux.Unlock()
	c.conn = newConn

	return nil
}

// Close destructs the connection.
func (c *ConnectionWrap) Close() {
	c.closeOnce.Do(c.close)
}

func (c *ConnectionWrap) close() {
	close(c.closeCh)
	conn := c.getConn()
	if conn != nil {
		conn.Close()
	}
	c.onClose()
}

func (c *ConnectionWrap) getConn() *nats_lib.Conn {
	c.connMux.RLock()
	defer c.connMux.RUnlock()
	return c.conn
}

// Servers returns list of currently connected servers.
func (c *ConnectionWrap) Servers() []string {
	return c.servers
}

// Publish method proxies to original method of nats.Conn
func (c *ConnectionWrap) Publish(subject string, payload []byte) error {
	return c.getConn().Publish(subject, payload)
}

// Subscribe method proxies to original method of nats.Conn
func (c *ConnectionWrap) Subscribe(subject string, handler nats_lib.MsgHandler) (*nats_lib.Subscription, error) {
	return c.getConn().Subscribe(subject, handler)
}

// Request method proxies to original method of nats.Conn
func (c *ConnectionWrap) Request(subject string, payload []byte, timeout time.Duration) (*nats_lib.Msg, error) {
	return c.getConn().Request(subject, payload, timeout)
}

// RequestWithContext method proxies to original method of nats.Conn
func (c *ConnectionWrap) RequestWithContext(ctx context.Context, subj string, data []byte) (*nats_lib.Msg, error) {
	return c.getConn().RequestWithContext(ctx, subj, data)
}

type dialer struct {
	dialer requests.DialContext
}

func (d *dialer) Dial(network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), nats_lib.DefaultTimeout)
	defer cancel()

	return d.dialer(ctx, network, address)
}
