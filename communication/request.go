package communication

type RequestType string

type RequestPacker interface {
	GetRequestType() RequestType
	CreateRequest() (requestPtr interface{})
	CreateResponse() (responsePtr interface{})
}

type RequestUnpacker interface {
	GetRequestType() RequestType
	CreateRequest() (messagePtr interface{})
	Handle(requestPtr interface{}) (responsePtr interface{}, err error)
}
