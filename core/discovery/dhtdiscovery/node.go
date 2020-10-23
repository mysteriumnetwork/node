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
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

// Node represents DHT server-client in P2P network.
type Node struct {
	libP2PNode       host.Host
	libP2PNodeCtx    context.Context
	libP2PNodeCancel context.CancelFunc

	bootstrapPeers []*peer.AddrInfo
}

// NewNode create an instance of DHT node.
func NewNode(listenAddress string, bootstrapPeerAddresses []string) (*Node, error) {
	// Parse and validate configuration
	listenAddr, err := multiaddr.NewMultiaddr(listenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DHT listen address. %w", err)
	}

	bootstrapPeers := make([]*peer.AddrInfo, len(bootstrapPeerAddresses))

	for i, peerAddress := range bootstrapPeerAddresses {
		peerAddr, err := multiaddr.NewMultiaddr(peerAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DHT peer address. %w", err)
		}

		if bootstrapPeers[i], err = peer.AddrInfoFromP2pAddr(peerAddr); err != nil {
			return nil, fmt.Errorf("failed to parse DHT peer info. %w", err)
		}
	}

	// Preparing config for libp2p Host. Other options can be added here.
	var config libp2p.Config
	if err = config.Apply(
		libp2p.ListenAddrs(listenAddr),
		libp2p.FallbackDefaults,
	); err != nil {
		return nil, fmt.Errorf("failed to configure DHT node. %w", err)
	}

	// Prepare context which stops the libp2p host.
	ctx, ctxCancel := context.WithCancel(context.Background())

	// Constructs a new libp2p node.
	node, err := config.NewNode(ctx)
	if err != nil {
		ctxCancel()

		return nil, fmt.Errorf("failed to start DHT node. %w", err)
	}

	log.Info().Msgf("DHT node started on %s with ID=%s", node.Addrs(), node.ID())

	return &Node{
		libP2PNode:       node,
		libP2PNodeCtx:    ctx,
		libP2PNodeCancel: ctxCancel,
		bootstrapPeers:   bootstrapPeers,
	}, nil
}

// Start begins DHT bootstrapping process.
func (n *Node) Start() error {
	// Let's connect to the bootstrap nodes first. They will tell us about the other nodes in the network.
	for _, peerInfo := range n.bootstrapPeers {
		go n.connectToPeer(*peerInfo)
	}

	return nil
}

// Stop stops DHT node.
func (n *Node) Stop() {
	n.libP2PNodeCancel()
}

func (n *Node) connectToPeer(peerInfo peer.AddrInfo) {
	if err := n.libP2PNode.Connect(n.libP2PNodeCtx, peerInfo); err != nil {
		log.Warn().Err(err).Msgf("Failed to contact DHT peer %s", peerInfo.ID)

		return
	}

	log.Info().Msgf("Connection established with DHT peer: %v", peerInfo)
}
