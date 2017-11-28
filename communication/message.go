package communication

type MessageType string

type MessageListener struct {
	Message Unpacker
	Invoke  func()
}

type MessagePacker interface {
	GetMessageType() MessageType
	CreateMessage() (messagePtr interface{})
}

type MessageUnpacker struct {
	MessageType string
	Unpack      func([]byte) error
	Invoke      func() error
}
