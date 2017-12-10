package nats_dialog

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
)

func NewServer(address *nats_discovery.NatsAddress) *serverNats {
	return &serverNats{
		myAddress: address,
	}
}

const SERVER_LOG_PREFIX = "[NATS.Server] "

type serverNats struct {
	myAddress *nats_discovery.NatsAddress
}

func (server *serverNats) ServeDialogs(dialogHandler communication.DialogHandler) error {
	log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Connecting to: %#v", server.myAddress))
	err := server.myAddress.Connect()
	if err != nil {
		return fmt.Errorf("Failed to start my connection. %s", server.myAddress)
	}

	myReceiver := nats.NewReceiver(server.myAddress)
	contactSender := nats.NewSender(server.myAddress)

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		dialogHandler(contactSender, myReceiver)

		log.Info(SERVER_LOG_PREFIX, fmt.Sprintf("Dialog accepted from: '%s'", request.IdentityId))
		return &dialogCreateResponse{Accepted: true}, nil
	}

	subscribeError := myReceiver.Respond(&dialogCreateHandler{createDialog})
	return subscribeError
}

func (server *serverNats) Start() (err error) {
	return nil
}

func (server *serverNats) Stop() error {
	return nil
}
