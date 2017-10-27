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

	createDialog := newRequestHandler(func(request *dialogCreateRequest) *dialogCreateResponse {
		sender := &senderNats{
			connection:     server.connection,
			messageTopic:   string(request.IdentityId),
			timeoutRequest: server.timeoutRequest,
		}
		dialogHandler(sender, receiver)

		log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Dialog with '%s' established.", request.IdentityId))
		return &dialogCreateResponse{
			Accepted: true,
		}
	})

	subscribeError := receiver.Respond(ENDPOINT_DIALOG_CREATE, createDialog)
	return subscribeError
}

func (server *serverNats) Start() (err error) {
	server.connection, err = server.options.Connect()
	return err
}

func (server *serverNats) Stop() error {
	server.connection.Close()
	return nil
}

func newRequestHandler(callback func(request *dialogCreateRequest) *dialogCreateResponse) communication.RequestHandler {
	var request dialogCreateRequest

	return communication.RequestHandler{
		Request: &request,
		Invoke: func() communication.Packer {
			return callback(&request)
		},
	}
}
