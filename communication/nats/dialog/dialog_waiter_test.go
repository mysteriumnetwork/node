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

package dialog

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var _ communication.DialogWaiter = &dialogWaiter{}

func TestDialogWaiter_Factory(t *testing.T) {
	connection := nats.NewConnection("nats://far-server:4222")
	signer := &identity.SignerFake{}

	waiter := NewDialogWaiter(connection, "custom", signer)
	assert.NotNil(t, waiter)
	assert.Equal(t, "custom", waiter.topic)
	assert.Equal(t, signer, waiter.signer)
}

func TestDialogWaiter_Start(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	waiter := NewDialogWaiter(connection, "123456", &identity.SignerFake{})
	contact, err := waiter.Start()

	assert.NoError(t, err)
	assert.Equal(
		t,
		market.Contact{
			Type: "nats/v1",
			Definition: nats_discovery.ContactNATSV1{
				Topic:           "123456",
				BrokerAddresses: []string{"mockhost"},
			},
		},
		contact,
	)
}

func TestDialogWaiter_ServeDialogs(t *testing.T) {
	peerID := identity.FromAddress("0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea")

	connection := nats.StartConnectionMock()
	defer connection.Close()

	signer := &identity.SignerFake{}
	waiter, handler := dialogServe(connection, signer)
	defer waiter.Stop()

	dialogAsk(connection, `{
		"payload": {"peer_id":"0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"},
		"signature": "tl+WbYkJdXD5foaIP3bqVGFHfr6kdd5FzmJAmu1GdpINEnNR3bTto6wgEoke/Fpy4zsWOjrulDVfrc32f5ArTgA="
	}`)
	dialogInstance, err := dialogWait(handler)
	defer dialogInstance.Close()
	assert.NoError(t, err)
	assert.NotNil(t, dialogInstance)

	dialog, ok := dialogInstance.(*dialog)
	assert.True(t, ok)

	expectedCodec := NewCodecSecured(communication.NewCodecJSON(), signer, identity.NewVerifierIdentity(peerID))
	assert.Equal(
		t,
		nats.NewSender(connection, expectedCodec, "my-topic.0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"),
		dialog.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, expectedCodec, "my-topic.0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"),
		dialog.Receiver,
	)
}

func TestDialogWaiter_ServeDialogsTopicUUID(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	signer := &identity.SignerFake{}

	handler := &dialogHandler{
		dialogReceived: make(chan communication.Dialog),
	}

	waiter := NewDialogWaiter(connection, "my-topic", signer)

	err := waiter.ServeDialogs(handler)
	assert.NoError(t, err)

	go func() {
		dialogWait(handler)
	}()

	msg, err := connection.Request("my-topic.dialog-create", []byte(`{
		"payload": {
			"peer_id":"0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea",
			"version":"v1"
		},
		"signature": "tl+WbYkJdXD5foaIP3bqVGFHfr6kdd5FzmJAmu1GdpINEnNR3bTto6wgEoke/Fpy4zsWOjrulDVfrc32f5ArTgA="
	}`), 100*time.Millisecond)
	assert.NoError(t, err)

	var response struct {
		Payload dialogCreateResponse `json:"payload"`
	}

	err = json.Unmarshal(msg.Data, &response)
	assert.NoError(t, err)

	_, err = uuid.FromString(response.Payload.Topic)
	assert.NoError(t, err)
}

func TestDialogWaiter_ServeDialogsRejectInvalidSignature(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	signer := &identity.SignerFake{}
	waiter, handler := dialogServe(connection, signer)
	defer waiter.Stop()

	dialogAsk(connection, `{
		"payload": {"peer_id":"0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"},
		"signature": "malformed"
	}`)
	dialogInstance, err := dialogWait(handler)
	assert.EqualError(t, err, "dialog not received")
	assert.Nil(t, dialogInstance)
}

func TestDialogWaiter_ServeDialogsRejectConsumersUsingValidator(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	signer := &identity.SignerFake{}

	mockeDialogHandler := &dialogHandler{
		dialogReceived: make(chan communication.Dialog),
	}

	waiter := NewDialogWaiter(
		connection,
		"test-topic",
		signer,
		func(_ identity.Identity) error {
			return errors.New("expected error")
		},
	)

	err := waiter.ServeDialogs(mockeDialogHandler)
	assert.NoError(t, err)

	msg, err := connection.Request("test-topic.dialog-create", []byte(`{
		"payload": {"peer_id":"0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"},
		"signature": "tl+WbYkJdXD5foaIP3bqVGFHfr6kdd5FzmJAmu1GdpINEnNR3bTto6wgEoke/Fpy4zsWOjrulDVfrc32f5ArTgA="
	}`), 100*time.Millisecond)
	assert.NoError(t, err)

	assert.JSONEq(
		t,
		`{
			"payload":	{
				"reason":400,
				"reasonMessage":"Invalid Identity"
			},
			"signature":"c2lnbmVkeyJyZWFzb24iOjQwMCwicmVhc29uTWVzc2FnZSI6IkludmFsaWQgSWRlbnRpdHkifQ=="
		}`,
		string(msg.Data),
	)
}

func dialogServe(connection *nats.ConnectionMock, signer identity.Signer) (waiter *dialogWaiter, handler *dialogHandler) {
	waiter = &dialogWaiter{
		connection: connection,
		topic:      "my-topic",
		signer:     signer,
	}
	handler = &dialogHandler{
		dialogReceived: make(chan communication.Dialog),
	}

	err := waiter.ServeDialogs(handler)
	if err != nil {
		panic(err)
	}

	return waiter, handler
}

func dialogAsk(connection nats.Connection, payload string) {
	err := connection.Publish("my-topic.dialog-create", []byte(payload))
	if err != nil {
		panic(err)
	}
}

func dialogWait(handler *dialogHandler) (communication.Dialog, error) {
	select {
	case dialog := <-handler.dialogReceived:
		return dialog, nil

	case <-time.After(10 * time.Millisecond):
		return nil, errors.New("dialog not received")
	}
}

type dialogHandler struct {
	dialogReceived chan communication.Dialog
}

func (handler *dialogHandler) Handle(dialog communication.Dialog) error {
	handler.dialogReceived <- dialog
	return nil
}
