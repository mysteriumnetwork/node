package communication

type RequestType string

type RequestProducer interface {
	GetRequestType() RequestType
	NewResponse() (responsePtr interface{})
	Produce() (requestPtr interface{})
}

type RequestConsumer interface {
	GetRequestType() RequestType
	NewRequest() (messagePtr interface{})
	Consume(requestPtr interface{}) (responsePtr interface{}, err error)
}
