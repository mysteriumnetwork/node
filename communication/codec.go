package communication

type Codec interface {
	Pack(payloadPtr interface{}) (data []byte, err error)
	Unpack(data []byte, payloadPtr interface{}) error
}
