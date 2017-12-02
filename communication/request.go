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

type RequestUnpacker struct {
	RequestType   RequestType
	Invoke        func() error
	RequestUnpack func([]byte) error
	ResponsePack  func() ([]byte, error)
}
