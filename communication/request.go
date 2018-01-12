package communication

// RequestEndpoint is special type that describes unique message endpoint
type RequestEndpoint string

// RequestProducer represents instance which creates requests/responses of specific endpoint
type RequestProducer interface {
	GetRequestEndpoint() RequestEndpoint
	NewResponse() (responsePtr interface{})
	Produce() (requestPtr interface{})
}

// RequestConsumer represents instance which handles requests/responses of specific endpoint
type RequestConsumer interface {
	GetRequestEndpoint() RequestEndpoint
	NewRequest() (messagePtr interface{})
	Consume(requestPtr interface{}) (responsePtr interface{}, err error)
}
