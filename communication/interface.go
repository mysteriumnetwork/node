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
	Receive(unpacker MessageUnpacker) error
	Respond(requestType RequestType, handler RequestHandler) error
}

type Sender interface {
	Send(packer MessagePacker) error
	Request(requestType RequestType, request Packer, response Unpacker) error
}
