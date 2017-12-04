package communication

type MessageType string

type MessageProducer interface {
	GetMessageType() MessageType
	Produce() (messagePtr interface{})
}

type MessageHandler interface {
	GetMessageType() MessageType
	NewMessage() (messagePtr interface{})
	Handle(messagePtr interface{}) error
}
