package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"time"
)

type clientNats struct {
	myTopic        string
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (client *clientNats) CreateDialog(contact dto_discovery.Contact) (
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

	response, err := sender.Request(communication.DIALOG_CREATE, client.myTopic)
	if err != nil {
		return
	}
	if response != "OK" {
		err = fmt.Errorf("Dialog creation rejected: %s", response)
		return
	}

	receiver = &receiverNats{
		connection:   client.connection,
		messageTopic: client.myTopic + ".",
	}
	fmt.Printf("Dialog with contact created. topic=%s\n", contactTopic)
	return
}

func (client *clientNats) Start() (err error) {
	client.connection, err = client.options.Connect()
	return err
}

func (client *clientNats) Stop() error {
	client.connection.Close()
	return nil
}

func extractContactTopic(contact dto_discovery.Contact) (topic string, err error) {
	if contact.Type != CONTACT_NATS_V1 {
		return "", fmt.Errorf("Invalid contact type: %s", contact.Type)
	}
	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		return "", fmt.Errorf("Invalid contact definition: %#v", contact.Definition)
	}

	return contactNats.Topic, nil
}
