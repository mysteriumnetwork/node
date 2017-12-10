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

func (client *clientNats) CreateDialog(contact dto_discovery.Contact) (
	contactSender communication.Sender,
	receiver communication.Receiver,
	err error,
) {
	contactAddress, err := nats_discovery.NewAddressForContact(contact)
	if err != nil {
		return
	}

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Connecting to: %#v", contactAddress))
	err = contactAddress.Connect()
	if err != nil {
		err = fmt.Errorf("Failed to connect to: %#v. %s", contact, err)
		return
	}

	myReceiver := nats.NewReceiver(contactAddress)
	contactSender = nats.NewSender(contactAddress)

	response, err := contactSender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: client.myIdentity,
		},
	})
	if !response.(*dialogCreateResponse).Accepted {
		err = fmt.Errorf("Dialog creation rejected: %#v", response)
		return
	}

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Dialog established with: %#v", contact))
	return contactSender, myReceiver, err
}

func (client *clientNats) Start() (err error) {
	return nil
}

func (client *clientNats) Stop() error {
	return nil
}
