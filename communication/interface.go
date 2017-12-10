package communication

import dto_discovery "github.com/mysterium/node/service_discovery/dto"

type Server interface {
	ServeDialogs(handler func(Dialog)) error
	Stop() error
}

type Client interface {
	CreateDialog(contact dto_discovery.Contact) (Dialog, error)
}

type Receiver interface {
	Receive(handler MessageHandler) error
	Respond(handler RequestHandler) error
}

type Sender interface {
	Send(producer MessageProducer) error
	Request(producer RequestProducer) (responsePtr interface{}, err error)
}

type Dialog interface {
	Sender
	Receiver
	Close() error
}
