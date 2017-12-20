package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSenderInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}

func TestSenderNew(t *testing.T) {
	connection := &connectionFake{}

	assert.Equal(
		t,
		&senderNats{
			connection:     connection,
			codec:          communication.NewCodecJSON(),
			timeoutRequest: 500 * time.Millisecond,
			messageTopic:   "custom.",
		},
		NewSender(connection, "custom"),
	)
}
