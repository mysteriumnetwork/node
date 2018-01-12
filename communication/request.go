package communication

type RequestType string

type RequestProducer interface {
	GetRequestType() RequestType
	NewResponse() (responsePtr interface{})
	Produce() (request interface{})
}

type RequestConsumer interface {
	GetRequestType() RequestType
	NewRequest() (messagePtr interface{})
	Consume(request interface{}) (response interface{}, err error)
}
