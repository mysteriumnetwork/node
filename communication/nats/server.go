package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"

	log "github.com/cihub/seelog"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/pkg/errors"
)

const SERVER_LOG_PREFIX = "[NATS.Server] "

type serverNats struct {
	myIdentity     dto_discovery.Identity
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (server *serverNats) ServeDialogs(dialogHandler communication.DialogHandler) error {
	if server.connection == nil {
		return errors.New("Client is not started")
	}

	receiver := newReceiver(server.connection, identityToTopic(server.myIdentity), nil)

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		contact := newContact(request.IdentityId)
		sender, _ := newSender(server.connection, contact, server.timeoutRequest, nil)

		dialogHandler(sender, receiver)
		log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Dialog with '%#v' established.", contact))

		return &dialogCreateResponse{Accepted: true}, nil
	}

	subscribeError := receiver.Respond(&dialogCreateHandler{createDialog})
	return subscribeError
}

func (server *serverNats) GetContact() dto_discovery.Contact {
	return newContact(server.myIdentity)
}

func (server *serverNats) Start() (err error) {
	server.connection, err = server.options.Connect()
	return err
}

func (server *serverNats) Stop() error {
	server.connection.Close()
	return nil
}
