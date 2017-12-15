package communication

type MessageType string

type MessageProducer interface {
	GetMessageType() MessageType
	Produce() (messagePtr interface{})
}

type MessageConsumer interface {
	GetMessageType() MessageType
	NewMessage() (messagePtr interface{})
	Consume(messagePtr interface{}) error
}
