package nats

import (
	"github.com/dshearer/jobber/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"testing"
	"time"
)

func TestSenderInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}

func TestSenderNew(t *testing.T) {
	connection := &nats.Conn{}
	receiverContact := newContact(dto_discovery.Identity("123456"))

	sender, err := newSender(connection, receiverContact, 0, nil)
	assert.NoError(t, err)
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
