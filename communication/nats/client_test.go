package nats

import (
	"github.com/mysterium/node/communication"
	"testing"
)

func TestClientInterface(t *testing.T) {
	var _ communication.Client = &clientNats{}
}
