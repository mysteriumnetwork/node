package communication

type MessageType string

type MessageListener struct {
	Message Unpacker
	Invoke  func()
}
