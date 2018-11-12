/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"sync"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

type fakeDialog struct {
	peerID    identity.Identity
	sessionID session.ID

	closed bool
	sync.RWMutex
}

func (fd *fakeDialog) PeerID() identity.Identity {
	fd.RLock()
	defer fd.RUnlock()

	return fd.peerID
}

func (fd *fakeDialog) Close() error {
	fd.Lock()
	defer fd.Unlock()

	fd.closed = true
	return nil
}

func (fd *fakeDialog) Receive(consumer communication.MessageConsumer) error {
	return nil
}
func (fd *fakeDialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}

func (fd *fakeDialog) Send(producer communication.MessageProducer) error {
	return nil
}

func (fd *fakeDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return &session.CreateResponse{
			Success: true,
			Session: session.SessionDto{
				ID:     fd.sessionID,
				Config: []byte("{}"),
			},
		},
		nil
}
