package nats

import (
	"fmt"
	"github.com/mgutz/logxi/v1"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
)

const CLIENT_LOG_PREFIX = "[NATS.Client] "

type clientNats struct {
	myIdentity dto_discovery.Identity

	options    nats.Options
	connection *nats.Conn
}

func (client *clientNats) CreateDialog(contact dto_discovery.Contact) (
	contactSender communication.Sender,
	receiver communication.Receiver,
	err error,
) {
	myReceiver, err := client.listen()
	if err != nil {
		err = fmt.Errorf("Failed to start my channel. %s", err)
		return
	}

	contactSender, err = client.contactConnect(contact)
	if err != nil {
		err = fmt.Errorf("Failed to start contact %#v channel. %s", contact, err)
		return
	}

	response, err := contactSender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: client.myIdentity,
		},
	})
	if !response.(*dialogCreateResponse).Accepted {
		err = fmt.Errorf("Dialog creation rejected: %#v", response)
		return
	}

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Dialog created with: %#v\n", contact))
	return contactSender, myReceiver, err
}

func (client *clientNats) Start() (err error) {
	client.connection, err = client.options.Connect()
	return err
}

func (client *clientNats) Stop() error {
	client.connection.Close()
	return nil
}

func (client *clientNats) listen() (communication.Receiver, error) {
	topic := identityToTopic(client.myIdentity)

	receiver := newReceiver(client.connection, topic, communication.NewCodecJSON())
	return receiver, nil
}

func (client *clientNats) contactConnect(contact dto_discovery.Contact) (communication.Sender, error) {
	contactTopic, err := contactToTopic(contact)
	if err != nil {
		return nil, err
	}

	sender := newSender(client.connection, contactTopic)
	return sender, nil
}
