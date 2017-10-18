package communication

type BytesPayload struct {
	Data []byte
}

func (payload BytesPayload) Pack() (data []byte) {
	return payload.Data
}

func (payload *BytesPayload) Unpack(data []byte) {
	payload.Data = data
}

func BytesListener(callback func(*BytesPayload)) MessageListener {
	return func(messageData []byte) {
		var message BytesPayload
		message.Unpack(messageData)

		callback(&message)
	}
}

func BytesHandler(callback func(*BytesPayload) *BytesPayload) RequestHandler {
	return func(requestData []byte) []byte {
		var message BytesPayload
		message.Unpack(requestData)

		response := callback(&message)

		return response.Pack()
	}
}
