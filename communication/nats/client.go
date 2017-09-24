package nats

import (
	"fmt"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"time"
)

type clientNats struct {
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (client *clientNats) CreateDialog(contact dto_discovery.Contact) (
	sender communication.Sender,
	receiver communication.Receiver,
	err error,
) {
	myTopic := "consumer1"

	if contact.Type != CONTACT_NATS_V1 {
		err = fmt.Errorf("Invalid contact type: %s", contact.Type)
		return
	}
	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		err = fmt.Errorf("Invalid contact definition: %#v", contact.Definition)
		return
	}

	if err = client.Start(); err != nil {
		return
	}

	message, err := client.connection.Request(contactNats.Topic, []byte(myTopic), client.timeoutRequest)
	if err != nil {
		return
	}

	response := string(message.Data)
	if response != "OK" {
		err = fmt.Errorf("Dialog creation rejected: %s", response)
		return
	}

	sender = &senderNats{
		connection:     client.connection,
		timeoutRequest: client.timeoutRequest,
	}
	receiver = &receiverNats{
		connection: client.connection,
		myTopic:    myTopic,
	}
	fmt.Printf("Dialog with contact created. topic=%s\n", contactNats.Topic)
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
