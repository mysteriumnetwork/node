package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"

	log "github.com/cihub/seelog"
)

const SERVER_LOG_PREFIX = "[NATS.Server] "

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

	createDialog := communication.StringHandler(func(receiverTopic *communication.StringPayload) *communication.StringPayload {
		sender := &senderNats{
			connection:     server.connection,
			messageTopic:   server.myTopic,
			timeoutRequest: server.timeoutRequest,
		}
		dialogHandler(sender, receiver)

		log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Dialog with '%s' established.", receiverTopic))
		return &communication.StringPayload{"OK"}
	})

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
