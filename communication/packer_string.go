package communication

type StringPayload struct {
	Data string
}

func (payload StringPayload) Pack() ([]byte, error) {
	return []byte(payload.Data), nil
}

func (payload *StringPayload) Unpack(data []byte) error {
	payload.Data = string(data)
	return nil
}

func StringListener(listener func(*StringPayload)) MessageListener {
	var message StringPayload

	return MessageListener{
		Message: &message,
		Invoke: func() {
			listener(&message)
		},
	}
}

func StringHandler(handler func(*StringPayload) *StringPayload) RequestHandler {
	var request StringPayload

	return RequestHandler{
		Request: &request,
		Invoke: func() (response Packer) {
			return handler(&request)
		},
	}
}
