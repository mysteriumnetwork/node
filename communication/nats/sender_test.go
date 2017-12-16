package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/stretchr/testify/assert"
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
			connection:     address.GetConnection(),
			codec:          communication.NewCodecJSON(),
			timeoutRequest: 500 * time.Millisecond,
			messageTopic:   "custom.",
		},
		sender,
	)
}
