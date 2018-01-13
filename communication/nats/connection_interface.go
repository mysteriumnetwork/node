package nats

import (
	"github.com/nats-io/go-nats"
	"time"
)

// Connection represents is publish-subscriber instance which can deliver messages
type Connection interface {
	Publish(subject string, payload []byte) error
	Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
	Request(subject string, payload []byte, timeout time.Duration) (*nats.Msg, error)
	Close()
}
