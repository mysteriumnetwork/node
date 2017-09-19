package communication

type Channel interface {
	Start() error
	Stop() error

	Send(messageType MessageType, message string) error
	Receive(messageType MessageType, callback MessageHandler) error

	Request(requestType RequestType, request string) (response string, err error)
	Respond(requestType RequestType, callback RequestHandler) error
}

type MessageType string
type MessageHandler func(message string)

type RequestType string
type RequestHandler func(request string) (response string)
