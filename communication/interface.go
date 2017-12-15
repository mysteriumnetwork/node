package communication

import dto_discovery "github.com/mysterium/node/service_discovery/dto"

type DialogWaiter interface {
	ServeDialogs(sessionCreateConsumer RequestConsumer) error
	Stop() error
}

type DialogEstablisher interface {
	CreateDialog(contact dto_discovery.Contact) (Dialog, error)
}

type Dialog interface {
	Sender
	Receiver
	Close() error
}

type Receiver interface {
	Receive(consumer MessageConsumer) error
	Respond(consumer RequestConsumer) error
}

type Sender interface {
	Send(producer MessageProducer) error
	Request(producer RequestProducer) (responsePtr interface{}, err error)
}
