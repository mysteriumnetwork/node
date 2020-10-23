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

package dhtdiscovery

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Node represents DHT server-client in P2P network.
type Node struct {
	libP2PNode       host.Host
	libP2PNodeCancel context.CancelFunc
}

// NewNode create an instance of DHT node.
func NewNode(listenAddress string) (*Node, error) {
	listenAddr, err := multiaddr.NewMultiaddr(listenAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse DHT listen address. %w")
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	node, err := libp2p.New(ctx,
		func(cfg *libp2p.Config) error {
			fmt.Printf("==config: %#v\n", cfg)
			cfg.Insecure = true
			return nil
		},
		libp2p.ListenAddrs(listenAddr),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT node. %w", err)
	}

	return &Node{
		libP2PNode:       node,
		libP2PNodeCancel: ctxCancel,
	}, nil
}

// Start begins DHT bootstrapping process.
func (n *Node) Start() error {
	log.Info().Msgf("DHT node created on %s. We are %s", n.libP2PNode.Addrs(), n.libP2PNode.ID())
	return nil
}

// Stop stops DHT node.
func (n *Node) Stop() {
	n.libP2PNodeCancel()
}
