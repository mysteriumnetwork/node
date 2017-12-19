package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReceiverInterface(t *testing.T) {
	var _ communication.Receiver = &receiverNats{}
}

func TestReceiverNew(t *testing.T) {
	connection := &connectionFake{}

	assert.Equal(
		t,
		&receiverNats{
			connection:   connection,
			codec:        communication.NewCodecJSON(),
			messageTopic: "custom.",
		},
		NewReceiver(connection, "custom"),
	)
}
