package dialog

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDialogWaiter_Interface(t *testing.T) {
	var _ communication.DialogWaiter = &dialogWaiter{}
}

func TestDialogWaiter_Factory(t *testing.T) {
	address := discovery.NewAddress("custom", "nats://far-server:4222")
	signer := &identity.SignerFake{}

	waiter := NewDialogWaiter(address, signer)
	assert.NotNil(t, waiter)
	assert.Equal(t, address, waiter.myAddress)
	assert.Equal(t, signer, waiter.mySigner)
}

func TestDialogWaiter_ServeDialogs(t *testing.T) {
	peerId := identity.FromAddress("0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea")

	connection := nats.StartConnectionFake()
	defer connection.Close()

	signer := &identity.SignerFake{}
	waiter, handler := dialogServe(connection, signer)
	defer waiter.Stop()

	dialogAsk(connection, `{
		"payload": {"identity_id":"0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"},
		"signature": "tl+WbYkJdXD5foaIP3bqVGFHfr6kdd5FzmJAmu1GdpINEnNR3bTto6wgEoke/Fpy4zsWOjrulDVfrc32f5ArTgA="
	}`)
	dialogInstance, err := dialogWait(handler)
	defer dialogInstance.Close()
	assert.NoError(t, err)
	assert.NotNil(t, dialogInstance)

	dialogNats, ok := dialogInstance.(*dialog)
	assert.True(t, ok)

	expectedCodec := NewCodecSecured(communication.NewCodecJSON(), signer, identity.NewVerifierIdentity(peerId))
	assert.Equal(
		t,
		nats.NewSender(connection, expectedCodec, "my-topic.0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"),
		dialogNats.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, expectedCodec, "my-topic.0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"),
		dialogNats.Receiver,
	)
}

func TestDialogWaiter_ServeDialogsRejectInvalidSignature(t *testing.T) {
	connection := nats.StartConnectionFake()
	defer connection.Close()

	signer := &identity.SignerFake{}
	waiter, handler := dialogServe(connection, signer)
	defer waiter.Stop()

	dialogAsk(connection, `{
		"payload": {"identity_id":"0x28bf83df144ab7a566bc8509d1fff5d5470bd4ea"},
		"signature": "malformed"
	}`)
	dialogInstance, err := dialogWait(handler)
	assert.EqualError(t, err, "dialog not received")
	assert.Nil(t, dialogInstance)
}

func dialogServe(connection nats.Connection, mySigner identity.Signer) (waiter *dialogWaiter, handler *dialogHandler) {
	myTopic := "my-topic"
	waiter = &dialogWaiter{
		myAddress: discovery.NewAddressWithConnection(connection, myTopic),
		mySigner:  mySigner,
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
