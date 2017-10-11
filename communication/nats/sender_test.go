package nats

import (
	"github.com/mysterium/node/communication"
	"testing"
)

func TestSenderInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}
