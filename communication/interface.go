/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package communication

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

// DialogWaiter defines server which:
//   - waits and servers incoming dialog requests
//   - negotiates with Dialog initiator
//   - finally creates Dialog, when it is accepted
type DialogWaiter interface {
	Start() (market.Contact, error)
	Stop() error
	ServeDialogs(DialogHandler) error
}

// DialogHandler defines how to handle incoming Dialog
type DialogHandler interface {
	Handle(Dialog) error
}

// DialogEstablisher interface defines client which:
//   - initiates Dialog requests to network
//   - creates Dialog, when it is negotiated
type DialogEstablisher interface {
	EstablishDialog(peerID identity.Identity, peerContact market.Contact) (Dialog, error)
}

// Dialog represent established connection between 2 peers in network.
// Enables bidirectional communication with another peer.
type Dialog interface {
	PeerID() identity.Identity
	Sender
	Receiver
	Close() error
}

// Receiver represents interface for:
//   - listening for asynchronous messages
//   - listening and serving HTTP-like requests
type Receiver interface {
	Receive(consumer MessageConsumer) error
	Respond(consumer RequestConsumer) error
	Unsubscribe() error
}

// Sender represents interface for:
//   - sending asynchronous messages
//   - sending and HTTP-like request and waiting for response
type Sender interface {
	Send(producer MessageProducer) error
	Request(producer RequestProducer) (responsePtr interface{}, err error)
}
