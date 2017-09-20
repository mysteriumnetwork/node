package communication

type Server interface {
	Start() error
	Stop() error

	Receive(messageType MessageType, callback MessageHandler) error
	Respond(requestType RequestType, callback RequestHandler) error
}

type Client interface {
	Start() error
	Stop() error

	Send(messageType MessageType, message string) error
	Request(requestType RequestType, request string) (response string, err error)
}

type MessageType string
type MessageHandler func(message string)

type RequestType string
type RequestHandler func(request string) (response string)
