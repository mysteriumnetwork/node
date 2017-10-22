package communication

type RequestType string

type RequestHandler struct {
	Request Unpacker
	Invoke  func() (response Packer)
}
