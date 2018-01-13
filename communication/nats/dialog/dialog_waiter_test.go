package dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDialogWaiter_Interface(t *testing.T) {
	var _ communication.DialogWaiter = &dialogWaiter{}
}

func TestDialogWaiter_Factory(t *testing.T) {
	address := discovery.NewAddress("custom", "nats://127.0.0.1:4222")
	signer := &identity.SignerFake{}

	waiter := NewDialogWaiter(address, signer)
	assert.NotNil(t, waiter)
	assert.Equal(t, address, waiter.myAddress)
	assert.Equal(t, signer, waiter.mySigner)
}

func TestDialogWaiter_newDialogToContact(t *testing.T) {
	connection := nats.NewConnectionFake()
	signer := &identity.SignerFake{}
	contactIdentity := identity.FromAddress("customer1")

	waiter := &dialogWaiter{
		myAddress: discovery.NewAddressWithConnection(connection, "provider1"),
		mySigner:  signer,
	}
	dialog := waiter.newDialogToContact(contactIdentity)

	expectedCodec := NewCodecSecured(communication.NewCodecJSON(), signer, identity.NewVerifierIdentity(contactIdentity))
	assert.Equal(
		t,
		nats.NewSender(connection, expectedCodec, "provider1.customer1"),
		dialog.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, expectedCodec, "provider1.customer1"),
		dialog.Receiver,
	)
}
