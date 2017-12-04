package communication

type RequestType string

type RequestProducer interface {
	GetRequestType() RequestType
	NewResponse() (responsePtr interface{})
	Produce() (requestPtr interface{})
}

type RequestHandler interface {
	GetRequestType() RequestType
	NewRequest() (messagePtr interface{})
	Handle(requestPtr interface{}) (responsePtr interface{}, err error)
}
