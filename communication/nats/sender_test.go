package nats

import (
	"github.com/dshearer/jobber/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	"testing"
	"time"
)

func TestSenderInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}

func TestSenderNew(t *testing.T) {
	address := nats_discovery.NewAddress("far-proxy", 1234, "custom")

	sender := NewSender(address)
	assert.Equal(
		t,
		&senderNats{
			connection:     nil,
			codec:          communication.NewCodecJSON(),
			timeoutRequest: 500 * time.Millisecond,
			messageTopic:   "custom.",
		},
		sender,
	)
}
