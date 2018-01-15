package communication

// RequestEndpoint is special type that describes unique message endpoint
type RequestEndpoint string

// RequestProducer represents instance which creates requests/responses of specific endpoint
type RequestProducer interface {
	GetRequestEndpoint() RequestEndpoint
	NewResponse() (responsePtr interface{})
	Produce() (request interface{})
}

// RequestConsumer represents instance which handles requests/responses of specific endpoint
type RequestConsumer interface {
	GetRequestEndpoint() RequestEndpoint
	NewRequest() (messagePtr interface{})
	Consume(request interface{}) (response interface{}, err error)
}
