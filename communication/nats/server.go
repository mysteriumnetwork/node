package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

const SERVER_LOG_PREFIX = "[NATS.Server] "

type serverNats struct {
	myTopic        string
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (server *serverNats) ServeDialogs(dialogHandler communication.DialogHandler) error {
	if server.connection == nil {
		return errors.New("Client is not started")
	}

	receiver := &receiverNats{
		connection:   server.connection,
		messageTopic: server.myTopic + ".",
	}

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		sender := &senderNats{
			connection:     server.connection,
			messageTopic:   string(request.IdentityId),
			timeoutRequest: server.timeoutRequest,
		}
		dialogHandler(sender, receiver)

		log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Dialog with '%s' established.", request.IdentityId))
		return &dialogCreateResponse{Accepted: true}, nil
	}

	subscribeError := receiver.Respond(&dialogCreateHandler{createDialog})
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
