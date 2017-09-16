package communication

type CommunicationsChannel interface {
	Start() error
	Stop() error
	Send(messageType MessageType, messagePayload string) error
	Receive(messageType MessageType, callback PayloadHandler) error
}

type PayloadHandler func(payload string)
