package nats_dialog

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewDialogEstablisher(identity identity.Identity) *dialogEstablisher {
	return &dialogEstablisher{
		myIdentity: identity,
		myCodec:    communication.NewCodecJSON(),
		contactAddressFactory: func(contact dto_discovery.Contact) (*nats_discovery.NatsAddress, error) {
			address, err := nats_discovery.NewAddressForContact(contact)
			if err == nil {
				err = address.Connect()
			}

			return address, err
		},
	}
}

const establisherLogPrefix = "[NATS.DialogEstablisher] "

type dialogEstablisher struct {
	myIdentity            identity.Identity
	myCodec               communication.Codec
	contactAddressFactory func(contact dto_discovery.Contact) (*nats_discovery.NatsAddress, error)
}

func (establisher *dialogEstablisher) CreateDialog(contact dto_discovery.Contact) (communication.Dialog, error) {
	var dialog *dialog

	log.Info(establisherLogPrefix, fmt.Sprintf("Connecting to: %#v", contact))
	contactAddress, err := establisher.contactAddressFactory(contact)
	if err != nil {
		return dialog, fmt.Errorf("Failed to connect to: %#v. %s", contact, err)
	}

	contactSender := nats.NewSender(contactAddress.GetConnection(), establisher.myCodec, contactAddress.GetTopic())
	response, err := contactSender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: establisher.myIdentity.Address,
		},
	})
	if err != nil {
		return dialog, fmt.Errorf("Dialog creation error. %s", err)
	}
	if response.(*dialogCreateResponse).Reason != 200 {
		return dialog, fmt.Errorf("Dialog creation rejected. %#v", response)
	}

	dialog = establisher.newDialogToContact(contactAddress)
	log.Info(establisherLogPrefix, fmt.Sprintf("Dialog established with: %#v", contact))

	return dialog, nil
}

func (establisher *dialogEstablisher) newDialogToContact(contactAddress *nats_discovery.NatsAddress) *dialog {
	subTopic := contactAddress.GetTopic() + "." + establisher.myIdentity.Address

	return &dialog{
		Sender:   nats.NewSender(contactAddress.GetConnection(), establisher.myCodec, subTopic),
		Receiver: nats.NewReceiver(contactAddress.GetConnection(), establisher.myCodec, subTopic),
	}
}
