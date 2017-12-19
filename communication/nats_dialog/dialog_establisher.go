package nats_dialog

import (
	"fmt"
	"github.com/mgutz/logxi/v1"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewDialogEstablisher(identity dto_discovery.Identity) *dialogEstablisher {
	return &dialogEstablisher{
		myIdentity:            identity,
		contactAddressFactory: nats_discovery.NewAddressForContact,
	}
}

const establisherLogPrefix = "[NATS.DialogEstablisher] "

type dialogEstablisher struct {
	myIdentity            dto_discovery.Identity
	contactAddressFactory func(contact dto_discovery.Contact) (*nats_discovery.NatsAddress, error)
}

func (establisher *dialogEstablisher) CreateDialog(contact dto_discovery.Contact) (communication.Dialog, error) {
	log.Info(establisherLogPrefix, fmt.Sprintf("Connecting to: %#v", contact))
	contactAddress, err := establisher.contactAddressFactory(contact)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to: %#v. %s", contact, err)
	}

	contactSender := nats.NewSender(contactAddress.GetConnection(), contactAddress.GetTopic())
	response, err := contactSender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: establisher.myIdentity,
		},
	})
	if err != nil || response.(*dialogCreateResponse).Reason != 200 {
		return nil, fmt.Errorf("Dialog creation rejected: %s", response)
	}

	dialog := establisher.newDialogToContact(contactAddress)
	log.Info(establisherLogPrefix, fmt.Sprintf("Dialog established with: %#v", contact))

	return dialog, err
}

func (establisher *dialogEstablisher) newDialogToContact(contactAddress *nats_discovery.NatsAddress) *dialog {
	subTopic := contactAddress.GetTopic() + "." + string(establisher.myIdentity)

	return &dialog{
		Sender:   nats.NewSender(contactAddress.GetConnection(), subTopic),
		Receiver: nats.NewReceiver(contactAddress.GetConnection(), subTopic),
	}
}
