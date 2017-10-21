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
	return func(requestData []byte) []byte {
		var message BytesPayload
		err := message.Unpack(requestData)
		if err != nil {
			panic(err)
		}

		response := callback(&message)

		responseData, err := response.Pack()
		if err != nil {
			panic(err)
		}
		return responseData
	}
}
