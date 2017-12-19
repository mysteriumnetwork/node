package nats_dialog

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/service_discovery/dto"
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

func (waiter *dialogWaiter) ServeDialogs(sessionCreateConsumer communication.RequestConsumer) error {
	log.Info(waiterLogPrefix, fmt.Sprintf("Connecting to: %#v", waiter.myAddress))
	err := waiter.myAddress.Connect()
	if err != nil {
		return fmt.Errorf("Failed to start my connection. %s", waiter.myAddress)
	}

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		if request.IdentityId == "" {
			return &responseInvalidIdentity, nil
		}

		dialog := waiter.newDialog(request.IdentityId)
		log.Info(waiterLogPrefix, fmt.Sprintf("Dialog accepted from: '%s'", request.IdentityId))

		dialog.Respond(sessionCreateConsumer)

		return &responseOK, nil
	}

	myReceiver := nats.NewReceiver(waiter.myAddress.GetConnection(), waiter.myAddress.GetTopic())
	subscribeError := myReceiver.Respond(&dialogCreateConsumer{createDialog})

	return subscribeError
}

func (waiter *dialogWaiter) newDialog(identity dto.Identity) *dialog {
	dialogAddress := nats_discovery.NewAddressNested(waiter.myAddress, string(identity))

	return &dialog{
		Sender:   nats.NewSender(dialogAddress.GetConnection(), dialogAddress.GetTopic()),
		Receiver: nats.NewReceiver(dialogAddress.GetConnection(), dialogAddress.GetTopic()),
	}
}

func (waiter *dialogWaiter) Stop() error {
	waiter.myAddress.Disconnect()

	return nil
}
