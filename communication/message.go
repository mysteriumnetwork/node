package communication

type MessageType string

type MessageHandler interface {
	Type() MessageType
	Deliver(messageBody []byte)
}
