package nats

import (
	"github.com/mysterium/node/communication"
	"testing"
)

func TestServerInterface(t *testing.T) {
	var _ communication.Client = &clientNats{}
}
