package communication

type CommunicationsChannel interface {
	Start() error
	Stop() error
	Send(messageType MessageType, messagePayload string) error
}
