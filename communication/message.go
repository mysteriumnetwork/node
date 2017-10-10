package communication

type MessageType string

type MessageProducer interface {
	MessageType() MessageType
	Produce() []byte
}

type MessageConsumer interface {
	MessageType() MessageType
	Consume(messageBody []byte)
}
