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
	"github.com/mysteriumnetwork/node/identity"
)

// Context represents request context.
type Context interface {
	// Request returns message with data bytes.
	Request() *Message

	// Error allows to return error which will be seen for peer.
	Error(err error) error

	// OkWithReply indicates that request was handled successfully and returns reply with given message.
	OkWithReply(msg *Message) error

	// Ok indicates that request was handled successfully
	OK() error

	// PeerID returns peer identity used to authenticate P2P channel
	PeerID() identity.Identity
}

type defaultContext struct {
	req         *Message
	res         *Message
	publicError error
	peerID      identity.Identity
}

func (d *defaultContext) Request() *Message {
	return d.req
}

func (d *defaultContext) Error(err error) error {
	d.publicError = err
	return nil
}

func (d *defaultContext) OkWithReply(msg *Message) error {
	d.res = msg
	return nil
}

func (d *defaultContext) OK() error {
	return nil
}

func (d *defaultContext) PeerID() identity.Identity {
	return d.peerID
}
