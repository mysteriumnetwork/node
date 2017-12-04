package nats

import (
	"fmt"
	"github.com/mgutz/logxi/v1"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
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
	contactTopic, err := extractContactTopic(contact)
	if err != nil {
		return
	}

	if err = client.Start(); err != nil {
		return
	}

	sender = &senderNats{
		connection:     client.connection,
		timeoutRequest: client.timeoutRequest,
		messageTopic:   contactTopic + ".",
	}

	resp, err := sender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: client.myIdentity,
		},
	})
	if !resp.(*dialogCreateResponse).Accepted {
		err = fmt.Errorf("Dialog creation rejected: %s", resp)
	}

	receiver = &receiverNats{
		connection:   client.connection,
		messageTopic: client.myTopic + ".",
	}

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
