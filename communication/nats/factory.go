package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

func NewService() communication.CommunicationsChannel {
	options := nats.GetDefaultOptions()
	options.Servers = []string{
		"nats://127.0.0.1:4222",
	}

	return &serviceNats{
		options:        options,
		timeoutRequest: 500 * time.Millisecond,
	}
}
