package communication

func NewCodecFake() *codecFake {
	return &codecFake{}
}

type codecFake struct {
	PackLastPayload interface{}
	packMock        []byte

	UnpackLastData []byte
	unpackMock     interface{}
}

func (codec *codecFake) MockPackResult(data []byte) {
	codec.packMock = data
}

func (codec *codecFake) MockUnpackResult(payload interface{}) {
	codec.unpackMock = payload
}

func (codec *codecFake) Pack(payloadPtr interface{}) ([]byte, error) {
	codec.PackLastPayload = payloadPtr

	return codec.packMock, nil
}

func (codec *codecFake) Unpack(data []byte, payloadPtr interface{}) error {
	codec.UnpackLastData = data

	payloadPtr = codec.unpackMock
	return nil
}
