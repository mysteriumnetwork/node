package nats_dialog

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
)

func NewDialogWaiter(address *nats_discovery.NatsAddress, signer identity.Signer) *dialogWaiter {
	return &dialogWaiter{
		myAddress: address,
		mySigner:  signer,
	}
}

const waiterLogPrefix = "[NATS.DialogWaiter] "

type dialogWaiter struct {
	myAddress *nats_discovery.NatsAddress
	mySigner  identity.Signer
}

func (waiter *dialogWaiter) ServeDialogs(sessionCreateConsumer communication.RequestConsumer) error {
	log.Info(waiterLogPrefix, fmt.Sprintf("Connecting to: %#v", waiter.myAddress))
	err := waiter.myAddress.Connect()
	if err != nil {
		return fmt.Errorf("Failed to start my connection. %s", waiter.myAddress)
	}

	createDialog := func(request dialogCreateRequest) (dialogCreateResponse, error) {
		if request.IdentityId == "" {
			return responseInvalidIdentity, nil
		}

		contactDialog := waiter.newDialogToContact(identity.FromAddress(request.IdentityId))
		log.Info(waiterLogPrefix, fmt.Sprintf("Dialog accepted from: '%s'", request.IdentityId))

		contactDialog.Respond(sessionCreateConsumer)

		return responseOK, nil
	}

	myCodec := NewCodecSecured(communication.NewCodecJSON(), waiter.mySigner, identity.NewVerifierSigned())
	myReceiver := nats.NewReceiver(waiter.myAddress.GetConnection(), myCodec, waiter.myAddress.GetTopic())
	subscribeError := myReceiver.Respond(&dialogCreateConsumer{createDialog})

	return subscribeError
}

func (waiter *dialogWaiter) newDialogToContact(contactIdentity identity.Identity) *dialog {
	subTopic := waiter.myAddress.GetTopic() + "." + contactIdentity.Address
	contactCodec := NewCodecSecured(
		communication.NewCodecJSON(),
		waiter.mySigner,
		identity.NewVerifierIdentity(contactIdentity),
	)

	return &dialog{
		Sender:   nats.NewSender(waiter.myAddress.GetConnection(), contactCodec, subTopic),
		Receiver: nats.NewReceiver(waiter.myAddress.GetConnection(), contactCodec, subTopic),
	}
}

func (waiter *dialogWaiter) Stop() error {
	waiter.myAddress.Disconnect()

	return nil
}
