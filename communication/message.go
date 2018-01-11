package communication

// MessageType is ppecial type that describes unique message endpoint
type MessageType string

// MessageProducer represents instance which creates messages of specific endpoint
type MessageProducer interface {
	GetMessageType() MessageType
	Produce() (messagePtr interface{})
}

// MessageConsumer represents instance which handles messages of specific endpoint
type MessageConsumer interface {
	GetMessageType() MessageType
	NewMessage() (messagePtr interface{})
	Consume(messagePtr interface{}) error
}
