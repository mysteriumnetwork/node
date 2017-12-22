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
	address := nats_discovery.NewAddress("127.0.0.1", 4222, "custom")
	waiter := NewDialogWaiter(address)

	assert.NotNil(t, waiter)
	assert.Equal(t, address, waiter.myAddress)
}

func TestDialogWaiter_newDialogToContact(t *testing.T) {
	connection := nats.NewConnectionFake()

	waiter := &dialogWaiter{
		myAddress: nats_discovery.NewAddressWithConnection(connection, "provider1"),
	}
	dialog := waiter.newDialogToContact(identity.FromAddress("customer1"))
	assert.Equal(
		t,
		nats.NewSender(connection, "provider1.customer1"),
		dialog.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, "provider1.customer1"),
		dialog.Receiver,
	)
}
