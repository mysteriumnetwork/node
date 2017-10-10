package communication

type MessageType string

type MessageProducer interface {
	ProduceMessage() []byte
}

type MessageConsumer interface {
	ConsumeMessage(messageBody []byte)
}
