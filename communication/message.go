package communication

type MessageType string

type MessagePacker interface {
	GetMessageType() MessageType
	CreateMessage() (messagePtr interface{})
}

type MessageUnpacker interface {
	GetMessageType() MessageType
	CreateMessage() (messagePtr interface{})
	Handle(messagePtr interface{}) error
}
