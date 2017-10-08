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
	Receive(messageType MessageType, callback MessageHandler) error
	Respond(requestType RequestType, callback RequestHandler) error
}

type Sender interface {
	Send(messageType MessageType, message []byte) error
	Request(requestType RequestType, request []byte) (response []byte, err error)
}
