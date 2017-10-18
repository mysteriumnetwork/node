package communication

type StringPayload struct {
	Data string
}

func (payload StringPayload) Pack() (data []byte) {
	return []byte(payload.Data)
}

func (payload *StringPayload) Unpack(data []byte) {
	payload.Data = string(data)
}

func StringListener(listener func(*StringPayload)) MessageListener {
	return func(messageData []byte) {
		var message StringPayload
		message.Unpack(messageData)

		listener(&message)
	}
}

func StringHandler(handler func(*StringPayload) *StringPayload) RequestHandler {
	return func(requestData []byte) []byte {
		var request StringPayload
		request.Unpack(requestData)

		response := handler(&request)

		return response.Pack()
	}
}
