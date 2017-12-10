package nats

import (
	"github.com/dshearer/jobber/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"testing"
	"time"
)

func TestSenderInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}

func TestSenderNew(t *testing.T) {
	connection := &nats.Conn{}

	sender := newSender(connection, "123456")
	assert.Equal(
		t,
		&senderNats{
			connection:     connection,
			codec:          communication.NewCodecJSON(),
			timeoutRequest: 500 * time.Millisecond,
			messageTopic:   "123456.",
		},
		sender,
	)
}
