package communication

type RequestType string

type RequestHandler struct {
	Request Unpacker
	Invoke  func() (response Packer)
}

type RequestPacker struct {
	RequestType    RequestType
	RequestPack    func() ([]byte, error)
	ResponseUnpack func([]byte) error
}
