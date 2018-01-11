package communication

// RequestType is ppecial type that describes unique message endpoint
type RequestType string

// RequestProducer represents instance which creates requests/responses of specific endpoint
type RequestProducer interface {
	GetRequestType() RequestType
	NewResponse() (responsePtr interface{})
	Produce() (requestPtr interface{})
}

// RequestConsumer represents instance which handles requests/responses of specific endpoint
type RequestConsumer interface {
	GetRequestType() RequestType
	NewRequest() (messagePtr interface{})
	Consume(requestPtr interface{}) (responsePtr interface{}, err error)
}
