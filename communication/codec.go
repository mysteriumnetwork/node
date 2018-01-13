package communication

// Codec interface defines how communication payload messages are
// encoded/decoded forward & backward
// before sending via communication Sender/Receiver
type Codec interface {
	Pack(payloadPtr interface{}) (data []byte, err error)
	Unpack(data []byte, payloadPtr interface{}) error
}
