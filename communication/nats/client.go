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
	options        nats.Options
	timeoutRequest time.Duration

	connection *nats.Conn
}

func (client *clientNats) CreateDialog(contact dto_discovery.Contact) (
	sender communication.Sender,
	receiver communication.Receiver,
	err error,
) {
	if client.connection == nil {
		err = errors.New("Client is not started")
		return
	}

	sender, err = newSender(client.connection, contact, client.timeoutRequest, nil)

	response, err := sender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: client.myIdentity,
		},
	})
	if !response.(*dialogCreateResponse).Accepted {
		err = fmt.Errorf("Dialog creation rejected: %#v", response)
		return
	}

	receiver = newReceiver(client.connection, identityToTopic(client.myIdentity), nil)

	log.Info(CLIENT_LOG_PREFIX, fmt.Sprintf("Dialog created with: %#v\n", contact))
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
