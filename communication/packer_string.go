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
	return func(requestData []byte) []byte {
		var request StringPayload
		err := request.Unpack(requestData)
		if err != nil {
			panic(err)
		}

		response := handler(&request)

		responseData, err := response.Pack()
		if err != nil {
			panic(err)
		}
		return responseData
	}
}
