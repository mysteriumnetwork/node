package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type serviceNats struct {
	options    nats.Options
	connection *nats.Conn
}

func (service *serviceNats) Start() (err error) {
	service.connection, err = service.options.Connect()
	return err
}

func (service *serviceNats) Stop() error {
	service.connection.Close()
	return nil
}

func (service *serviceNats) Send(messageType communication.MessageType, payload string) error {
	return service.connection.Publish(string(messageType), []byte(payload))
}

func (service *serviceNats) Receive(messageType communication.MessageType, callback communication.PayloadHandler) error {
	_, err := service.connection.Subscribe(string(messageType), func(message *nats.Msg) {
		callback(string(message.Data))
	})
	return err
}

func (service *serviceNats) ReceiveSync(messageType communication.MessageType) (string, error) {
	subscription, err := service.connection.SubscribeSync(string(messageType))
	if err != nil {
		return "", err
	}

	message, err := subscription.NextMsg(10 * time.Second)
	if err != nil {
		return "", err
	}

	return string(message.Data), nil
}
