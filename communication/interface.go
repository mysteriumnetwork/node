package communication

import dto_discovery "github.com/mysterium/node/service_discovery/dto"

type Server interface {
	ServeDialogs(callback DialogHandler) error
	Stop() error
}

type DialogHandler func(Sender, Receiver)

type Client interface {
	CreateDialog(contact dto_discovery.ContactDefinition) (Sender, Receiver, error)
	Stop() error
}

type Receiver interface {
	Receive(messageType MessageType, consumer MessageUnpacker) error
	Respond(requestType RequestType, consumer RequestConsumer) error
}

type Sender interface {
	Send(messageType MessageType, producer MessagePacker) error
	Request(requestType RequestType, request MessagePacker, response MessageUnpacker) error
}
