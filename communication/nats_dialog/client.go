package nats_dialog

import (
	"fmt"
	"github.com/mgutz/logxi/v1"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewClient(identity dto_discovery.Identity) *clientNats {
	return &clientNats{
		myIdentity: identity,
	}
}

const CLIENT_LOG_PREFIX = "[NATS.Client] "

type clientNats struct {
	myIdentity dto_discovery.Identity
}

func (client *clientNats) CreateDialog(contact dto_discovery.Contact) (communication.Dialog, error) {
	contactAddress, err := nats_discovery.NewAddressForContact(contact)
	if err != nil {
		return nil, err
	}

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Connecting to: %#v", contactAddress))
	err = contactAddress.Connect()
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to: %#v. %s", contact, err)
	}

	dialog := &dialog{nats.NewSender(contactAddress), nats.NewReceiver(contactAddress)}
	response, err := dialog.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: client.myIdentity,
		},
	})
	if err != nil || !response.(*dialogCreateResponse).Accepted {
		return nil, fmt.Errorf("Dialog creation rejected: %s", err)
	}

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Dialog established with: %#v", contact))
	return dialog, err
}
