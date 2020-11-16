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

package p2p

import (
	"fmt"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/mapping"
)

// Listener knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Listener interface {
	// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
	// to channelHandlers
	Listen(providerID identity.Identity, serviceType string, channelHandler func(ch Channel)) (func(), error)

	// GetContacts returns contracts which is later can be added to proposal contacts definition so consumer can
	// know how to connect to this p2p listener.
	GetContacts(serviceType, providerID string) []market.Contact
}

func NewListener(brokerConn nats.Connection, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, providerPinger natProviderPinger, portPool port.ServicePortSupplier, portMapper mapping.PortMapper, address address) Listener {
	return &listener{
		nats: &listenerNATS{
			brokerConn:     brokerConn,
			pendingConfigs: map[PublicKey]p2pConnectConfig{},
			ipResolver:     ipResolver,
			signer:         signer,
			verifier:       verifier,
			portPool:       portPool,
			providerPinger: providerPinger,
			portMapper:     portMapper,
		},
		http: &listenerHTTP{
			addressf:       address,
			pendingConfigs: map[PublicKey]p2pConnectConfig{},
			ipResolver:     ipResolver,
			signer:         signer,
			verifier:       verifier,
			portPool:       portPool,
			providerPinger: providerPinger,
			portMapper:     portMapper,
		},
	}
}

type listener struct {
	nats *listenerNATS
	http *listenerHTTP
}

func (l *listener) Listen(providerID identity.Identity, serviceType string, channelHandler func(ch Channel)) (func(), error) {
	fNATS, err := l.nats.Listen(providerID, serviceType, channelHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for incoming NATS connections: %w", err)
	}

	fHTTP, err := l.http.Listen(providerID, serviceType, channelHandler)
	if err != nil {
		fNATS()
		return nil, fmt.Errorf("failed to listen for incoming HTTP connections: %w", err)
	}

	return func() {
		fNATS()
		fHTTP()
	}, nil
}

func (l *listener) GetContacts(serviceType, providerID string) []market.Contact {
	contacts := l.nats.GetContacts(serviceType, providerID)
	for _, c := range l.http.GetContacts(serviceType, providerID) {
		contacts = append(contacts, c)
	}

	return contacts
}
