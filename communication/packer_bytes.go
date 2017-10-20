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
	return func(messageData []byte) {
		var message BytesPayload
		err := message.Unpack(messageData)
		if err != nil {
			panic(err)
		}

		callback(&message)
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
