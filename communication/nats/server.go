package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type serverNats struct {
	myTopic        string
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (server *serverNats) ServeDialogs(dialogHandler communication.DialogHandler) error {
	if err := server.Start(); err != nil {
		return err
	}

	receiver := &receiverNats{
		connection:   server.connection,
		messageTopic: server.myTopic + ".",
	}

	createDialog := func(request []byte) (response []byte) {
		receiverTopic := request
		fmt.Printf("Dialog requested. topic=%s\n", receiverTopic)

		sender := &senderNats{
			connection:     server.connection,
			messageTopic:   server.myTopic,
			timeoutRequest: server.timeoutRequest,
		}
		dialogHandler(sender, receiver)

		return []byte("OK")
	}

	return receiver.Respond(communication.DIALOG_CREATE, createDialog)
}

func (server *serverNats) Start() (err error) {
	server.connection, err = server.options.Connect()
	return err
}

func (server *serverNats) Stop() error {
	server.connection.Close()
	return nil
}
