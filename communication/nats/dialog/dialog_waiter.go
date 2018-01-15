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
func NewDialogWaiter(address *discovery.AddressNATS, signer identity.Signer) *dialogWaiter {
	return &dialogWaiter{
		myAddress: address,
		mySigner:  signer,
	}
}

const waiterLogPrefix = "[NATS.DialogWaiter] "

type dialogWaiter struct {
	myAddress *discovery.AddressNATS
	mySigner  identity.Signer
}

func (waiter *dialogWaiter) ServeDialogs(dialogHandler communication.DialogHandler) error {
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
		err = dialogHandler(contactDialog)
		if err != nil {
			log.Error(waiterLogPrefix, fmt.Sprintf("Failed dialog from: '%s'. %s", request.IdentityId, err))
			return &responseInternalError, nil
		}

		log.Info(waiterLogPrefix, fmt.Sprintf("Accepted dialog from: '%s'", request.IdentityId))
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
