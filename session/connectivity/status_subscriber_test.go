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

package connectivity

import (
	"testing"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestStatusSubscriber_AddsNewEntry(t *testing.T) {
	storage := &mockStatusStorage{}
	subscriber := NewStatusSubscriber(storage)
	dialog := &mockDialog{
		peerAddress: "p1",
		msg: &StatusMessage{
			SessionID:  "s1",
			StatusCode: StatusConnectionOk,
			Message:    "OK",
		},
	}

	subscriber.Subscribe(dialog)

	assert.Equal(t, dialog.peerAddress, storage.addedEntry.PeerID.Address)
	assert.Equal(t, dialog.msg.SessionID, storage.addedEntry.SessionID)
	assert.Equal(t, dialog.msg.StatusCode, storage.addedEntry.StatusCode)
	assert.Equal(t, dialog.msg.Message, storage.addedEntry.Message)
}

type mockDialog struct {
	peerAddress string
	msg         *StatusMessage
}

func (m *mockDialog) PeerID() identity.Identity {
	return identity.Identity{Address: m.peerAddress}
}

func (m *mockDialog) Send(producer communication.MessageProducer) error {
	return nil
}

func (m *mockDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return nil, nil
}

func (m *mockDialog) Receive(consumer communication.MessageConsumer) error {
	return consumer.Consume(m.msg)
}

func (m *mockDialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}

func (m *mockDialog) Unsubscribe() {
}

func (m *mockDialog) Close() error {
	return nil
}

type mockStatusStorage struct {
	addedEntry StatusEntry
}

func (m *mockStatusStorage) GetAllStatusEntries() []StatusEntry {
	return []StatusEntry{}
}

func (m *mockStatusStorage) AddStatusEntry(msg StatusEntry) {
	m.addedEntry = msg
}
