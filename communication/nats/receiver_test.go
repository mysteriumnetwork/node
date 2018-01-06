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
	codec := communication.NewCodecFake()

	assert.Equal(
		t,
		&receiverNats{
			connection:   connection,
			codec:        codec,
			messageTopic: "custom.",
		},
		NewReceiver(connection, codec, "custom"),
	)
}
