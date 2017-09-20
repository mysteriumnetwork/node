package nats

import (
	"github.com/mysterium/node/communication"
	"testing"
)

func TestReceiverInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}
