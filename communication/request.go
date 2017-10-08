package communication

type RequestType string

type RequestHandler func(request []byte) (response []byte)
