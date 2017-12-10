package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
)

const SERVER_LOG_PREFIX = "[NATS.Server] "

type serverNats struct {
	myIdentity dto_discovery.Identity

	options    nats.Options
	connection *nats.Conn
}

func (server *serverNats) ServeDialogs(dialogHandler communication.DialogHandler) error {
	myReceiver, err := server.listen()
	if err != nil {
		return fmt.Errorf("Failed to start my channel. %s", err)
	}

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		contactSender, err := server.contactConnect(request.IdentityId)
		if err != nil {
			log.Error(SERVER_LOG_PREFIX, fmt.Sprintf("Failed to start contact '%s' channel. %s", request.IdentityId, err))
			return nil, fmt.Errorf("Failed to connect to your channel")
		}

		dialogHandler(contactSender, myReceiver)
		log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Channel to contact '%s' established.", request.IdentityId))

		return &dialogCreateResponse{Accepted: true}, nil
	}

	subscribeError := myReceiver.Respond(&dialogCreateHandler{createDialog})
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

func (server *serverNats) listen() (communication.Receiver, error) {
	myTopic := identityToTopic(server.myIdentity)

	receiver := newReceiver(server.connection, myTopic, communication.NewCodecJSON())
	return receiver, nil
}

func (server *serverNats) contactConnect(contactIdentity dto_discovery.Identity) (communication.Sender, error) {
	contactTopic := identityToTopic(contactIdentity)

	sender := newSender(server.connection, contactTopic)
	return sender, nil
}

func (server *serverNats) GetContact() dto_discovery.Contact {
	return newContact(server.myIdentity)
}
