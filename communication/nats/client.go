package nats

import (
	"github.com/nats-io/go-nats"
	"time"
)

type clientNats struct {
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (client *clientNats) Start() (err error) {
	client.connection, err = client.options.Connect()
	return err
}

func (client *clientNats) Stop() error {
	client.connection.Close()
	return nil
}
