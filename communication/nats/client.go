package nats

import (
	"fmt"
	"github.com/mgutz/logxi/v1"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"time"
)

const CLIENT_LOG_PREFIX = "[NATS.Client] "

type clientNats struct {
	myIdentity     dto_discovery.Identity
	myTopic        string
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (client *clientNats) CreateDialog(contact dto_discovery.ContactDefinition) (
	sender communication.Sender,
	receiver communication.Receiver,
	err error,
) {
	if client.connection == nil {
		err = errors.New("Client is not started")
		return
	}

	contactTopic, err := extractContactTopic(contact)
	if err != nil {
		return
	}
	sender = newSender(client.connection, contactTopic, client.timeoutRequest, nil)

	response, err := sender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: client.myIdentity,
		},
	})
	if !response.(*dialogCreateResponse).Accepted {
		err = fmt.Errorf("Dialog creation rejected: %#v", response)
		return
	}

	receiver = newReceiver(client.connection, client.myTopic, nil)

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Dialog with '%s' created\n", contactTopic))
	return sender, receiver, err
}

func (client *clientNats) Start() (err error) {
	client.connection, err = client.options.Connect()
	return err
}

func (client *clientNats) Stop() error {
	client.connection.Close()
	return nil
}

func extractContactTopic(contact dto_discovery.ContactDefinition) (topic string, err error) {
	contactNats, ok := contact.(ContactNATSV1)
	if !ok {
		return "", fmt.Errorf("Invalid contact definition: %#v", contact)
	}

	return contactNats.Topic, nil
}
