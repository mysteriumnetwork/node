package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
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
