package communication

// MessageEndpoint is special type that describes unique message endpoint
type MessageEndpoint string

// MessageProducer represents instance which creates messages of specific endpoint
type MessageProducer interface {
	GetMessageEndpoint() MessageEndpoint
	Produce() (messagePtr interface{})
}

// MessageConsumer represents instance which handles messages of specific endpoint
type MessageConsumer interface {
	GetMessageEndpoint() MessageEndpoint
	NewMessage() (messagePtr interface{})
	Consume(messagePtr interface{}) error
}
