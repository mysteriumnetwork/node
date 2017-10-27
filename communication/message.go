package communication

type MessageType string

type MessageListener struct {
	Message Unpacker
	Invoke  func()
}

type MessagePacker struct {
	MessageType string
	Pack        func() ([]byte, error)
}

type MessageUnpacker struct {
	MessageType string
	Unpack      func([]byte) error
	Invoke      func() error
}
