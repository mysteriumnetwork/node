package nats

import (
	"github.com/nats-io/go-nats"
)

type serverNats struct {
	options nats.Options

	connection *nats.Conn
}

func (server *serverNats) Start() (err error) {
	server.connection, err = server.options.Connect()
	return err
}

func (server *serverNats) Stop() error {
	server.connection.Close()
	return nil
}
