package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReceiverInterface(t *testing.T) {
	var _ communication.Receiver = &receiverNats{}
}

func TestReceiverNew(t *testing.T) {
	address := nats_discovery.NewAddress("far-proxy", 1234, "custom")

	sender := NewReceiver(address)
	assert.Equal(
		t,
		&receiverNats{
			connection:   address.GetConnection(),
			codec:        communication.NewCodecJSON(),
			messageTopic: "custom.",
		},
		sender,
	)
}
