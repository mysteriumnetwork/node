package communication

type BytesPayload struct {
	Data []byte
}

func (payload BytesPayload) Pack() ([]byte, error) {
	return payload.Data, nil
}

func (payload *BytesPayload) Unpack(data []byte) error {
	payload.Data = data
	return nil
}

func BytesPacker(data []byte) *BytesPayload {
	return &BytesPayload{data}
}

func BytesListener(callback func(*BytesPayload)) MessageListener {
	var message BytesPayload

	return MessageListener{
		Message: &message,
		Invoke: func() {
			callback(&message)
		},
	}
}

func BytesHandler(callback func(*BytesPayload) *BytesPayload) RequestHandler {
	var request BytesPayload

	return RequestHandler{
		Request: &request,
		Invoke: func() Packer {
			return callback(&request)
		},
	}
}
