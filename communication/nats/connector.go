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
	"context"
	"net"
	"net/url"

	nats_lib "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/requests/resolver"
)

// BrokerConnector establishes new connections to NATS servers and handles reconnects.
type BrokerConnector struct {
	// resolveContext specifies the resolve function for doing custom DNS lookup.
	// If ResolveContext is nil, then the transport dials using package net.
	resolveContext resolver.ResolveContext

	dialer requests.DialContext
}

// NewBrokerConnector creates a new BrokerConnector.
func NewBrokerConnector(dialer requests.DialContext, resolveContext resolver.ResolveContext) *BrokerConnector {
	return &BrokerConnector{
		resolveContext: resolveContext,
		dialer:         dialer,
	}
}

func (b *BrokerConnector) resolveServers(serverURLs []*url.URL) ([]*url.URL, error) {
	if b.resolveContext == nil {
		return serverURLs, nil
	}

	for _, serverURL := range serverURLs {
		ctx, cancel := context.WithTimeout(context.Background(), nats_lib.DefaultTimeout)
		defer cancel()

		addrs, err := b.resolveContext(ctx, "tcp", serverURL.Host)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to resolve NATS server "%s"`, serverURL.Hostname())
		}

		cacheBrokerDNS(serverURL.Host, addrs)

		for _, addr := range addrs {
			serverURLResolved := *serverURL
			serverURLResolved.Host = addr
			serverURLs = append(serverURLs, &serverURLResolved)
		}
	}

	return serverURLs, nil
}

// Connect establishes a new connection to the broker(s).
func (b *BrokerConnector) Connect(serverURLs ...*url.URL) (Connection, error) {
	log.Debug().Msgf("Connecting to NATS servers: %v", serverURLs)

	serverURLs, err := b.resolveServers(serverURLs)
	if err != nil {
		return nil, err
	}

	servers := make([]string, len(serverURLs))
	for i, serverURL := range serverURLs {
		servers[i] = serverURL.String()
	}

	removeFirewallRule, err := firewall.AllowURLAccess(servers...)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to allow NATS servers "%v" in firewall`, servers)
	}

	conn, err := newConnection(b.dialer, servers...)
	if err != nil {
		return nil, err
	}

	if err := conn.Open(); err != nil {
		return nil, err
	}

	conn.onClose = removeFirewallRule

	return conn, nil
}

func cacheBrokerDNS(server string, addrs []string) {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		log.Warn().Msgf("Failed to parse broker address: %v", server)
		return
	}

	cacheAddrs := []string{}

	for _, addr := range addrs {
		if server == addr {
			continue
		}

		ip, _, err := net.SplitHostPort(addr)
		if err != nil {
			log.Warn().Msgf("Failed to parse broker address: %v", addr)
			continue
		}

		cacheAddrs = append(cacheAddrs, ip)
	}

	resolver.CacheDNSRecord(host, cacheAddrs)
}
