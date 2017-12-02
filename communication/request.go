package communication

type RequestType string

type RequestHandler struct {
	Request Unpacker
	Invoke  func() (response Packer)
}

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
