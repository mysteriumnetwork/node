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

package noop

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/promise"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
)

type fakeDialog struct {
	returnReceiveMessage interface{}
	returnError          error

	sendMutex   sync.RWMutex
	sendMessage interface{}
}

func (fd *fakeDialog) PeerID() identity.Identity {
	return identity.Identity{}
}

func (fd *fakeDialog) Close() error {
	return nil
}

func (fd *fakeDialog) Receive(consumer communication.MessageConsumer) error {
	if fd.returnError != nil {
		return fd.returnError
	}

	consumer.Consume(fd.returnReceiveMessage)
	return nil
}
func (fd *fakeDialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}
func (fd *fakeDialog) Unsubscribe() {}

func (fd *fakeDialog) Send(producer communication.MessageProducer) error {
	fd.sendMutex.Lock()
	defer fd.sendMutex.Unlock()

	if fd.returnError != nil {
		return fd.returnError
	}

	fd.sendMessage = producer.Produce()
	return nil
}

func (fd *fakeDialog) getSendMessage() interface{} {
	fd.sendMutex.Lock()
	defer fd.sendMutex.Unlock()

	return fd.sendMessage
}

func (fd *fakeDialog) waitSendMessage() (interface{}, error) {
	for i := 0; i < 10; i++ {
		if message := fd.getSendMessage(); message != nil {
			return message, nil
		}
		time.Sleep(time.Millisecond)
	}
	return nil, errors.New("message was not sent")
}

func (fd *fakeDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return &promise.Response{Success: true}, nil
}
