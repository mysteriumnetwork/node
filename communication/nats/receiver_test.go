package nats

import (
	"github.com/magiconair/properties/assert"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	"testing"
)

func TestReceiverInterface(t *testing.T) {
	var _ communication.Receiver = &receiverNats{}
}

func TestReceiverNew(t *testing.T) {
	connection := nats_discovery.NewAddress("far-proxy", 1234, "custom")

	sender := newReceiver(connection)
	assert.Equal(
		t,
		&receiverNats{
			connection:   nil,
			codec:        communication.NewCodecJSON(),
			messageTopic: "custom.",
		},
		sender,
	)
}
