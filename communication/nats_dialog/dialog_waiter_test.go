package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDialogWaiter_Interface(t *testing.T) {
	var _ communication.DialogWaiter = &dialogWaiter{}
}

func TestDialogWaiter_Factory(t *testing.T) {
	address := nats_discovery.NewAddress("custom", "nats://127.0.0.1:4222",)
	waiter := NewDialogWaiter(address)

	assert.NotNil(t, waiter)
	assert.Equal(t, address, waiter.myAddress)
}

func TestDialogWaiter_newDialogToContact(t *testing.T) {
	connection := nats.NewConnectionFake()
	codec := communication.NewCodecJSON()

	waiter := &dialogWaiter{
		myAddress: nats_discovery.NewAddressWithConnection(connection, "provider1"),
		myCodec:   codec,
	}
	dialog := waiter.newDialogToContact(identity.FromAddress("customer1"))
	assert.Equal(
		t,
		nats.NewSender(connection, codec, "provider1.customer1"),
		dialog.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, codec, "provider1.customer1"),
		dialog.Receiver,
	)
}
