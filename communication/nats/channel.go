package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type serviceNats struct {
	options        nats.Options
	timeoutRequest time.Duration
	connection     *nats.Conn
}

func (service *serviceNats) Start() (err error) {
	service.connection, err = service.options.Connect()
	return err
}

func (service *serviceNats) Stop() error {
	service.connection.Close()
	return nil
}

func (service *serviceNats) Send(
	messageType communication.MessageType,
	message string,
) error {
	return service.connection.Publish(string(messageType), []byte(message))
}

func (service *serviceNats) Receive(
	messageType communication.MessageType,
	callback communication.MessageHandler,
) error {
	_, err := service.connection.Subscribe(string(messageType), func(message *nats.Msg) {
		callback(string(message.Data))
	})

	return err
}

func (service *serviceNats) Request(
	messageType communication.RequestType,
	request string,
) (response string, err error) {
	message, err := service.connection.Request(string(messageType), []byte(request), service.timeoutRequest)
	if err != nil {
		return
	}

	response = string(message.Data)
	return
}

func (service *serviceNats) Respond(
	messageType communication.RequestType,
	callback communication.RequestHandler,
) error {
	_, err := service.connection.Subscribe(string(messageType), func(message *nats.Msg) {
		response := callback(string(message.Data))
		service.connection.Publish(message.Reply, []byte(response))
	})

	return err
}
