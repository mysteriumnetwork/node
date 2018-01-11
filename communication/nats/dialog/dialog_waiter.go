package dialog

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
)

// NewDialogWaiter constructs new DialogWaiter which works thru NATS connection.
func NewDialogWaiter(address *discovery.NatsAddress, signer identity.Signer) *dialogWaiter {
	return &dialogWaiter{
		myAddress: address,
		mySigner:  signer,
	}
}

const waiterLogPrefix = "[NATS.DialogWaiter] "

type dialogWaiter struct {
	myAddress *discovery.NatsAddress
	mySigner  identity.Signer
}

func (waiter *dialogWaiter) ServeDialogs(sessionCreateConsumer communication.RequestConsumer) error {
	log.Info(waiterLogPrefix, fmt.Sprintf("Connecting to: %#v", waiter.myAddress))
	err := waiter.myAddress.Connect()
	if err != nil {
		return fmt.Errorf("failed to start my connection. %s", waiter.myAddress)
	}

	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		if request.IdentityId == "" {
			return &responseInvalidIdentity, nil
		}

		contactDialog := waiter.newDialogToContact(identity.FromAddress(request.IdentityId))
		log.Info(waiterLogPrefix, fmt.Sprintf("Dialog accepted from: '%s'", request.IdentityId))

		contactDialog.Respond(sessionCreateConsumer)

		return &responseOK, nil
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
