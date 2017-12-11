package nats_dialog

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
)

func NewDialogWaiter(address *nats_discovery.NatsAddress) *dialogWaiter {
	return &dialogWaiter{
		myAddress: address,
	}
}

const waiterLogPrefix = "[NATS.DialogWaiter] "

type dialogWaiter struct {
	myAddress *nats_discovery.NatsAddress
}

func (waiter *dialogWaiter) ServeDialogs(sessionCreateHandler communication.RequestHandler) error {
	log.Info(waiterLogPrefix, fmt.Sprintf("Connecting to: %#v", waiter.myAddress))
	err := waiter.myAddress.Connect()
	if err != nil {
		return fmt.Errorf("Failed to start my connection. %s", waiter.myAddress)
	}

	dialog := &dialog{nats.NewSender(waiter.myAddress), nats.NewReceiver(waiter.myAddress)}

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		dialog.Respond(sessionCreateHandler)

		log.Info(waiterLogPrefix, fmt.Sprintf("Dialog accepted from: '%s'", request.IdentityId))
		return &dialogCreateResponse{Accepted: true}, nil
	}

	subscribeError := dialog.Respond(&dialogCreateHandler{createDialog})
	return subscribeError
}

func (waiter *dialogWaiter) Stop() error {
	waiter.myAddress.Disconnect()

	return nil
}
