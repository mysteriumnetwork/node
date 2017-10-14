package communication

func StringPacker(message string) Packer {
	return func() []byte {
		return []byte(message)
	}
}

func StringUnpacker(message *string) Unpacker {
	return func(data []byte) {
		*message = string(data)
	}
}

func StringListener(listener func(string)) MessageListener {
	return func(messageData []byte) {
		message := string(messageData)
		listener(message)
	}
}

func StringHandler(handler func(string) string) RequestHandler {
	return func(requestData []byte) []byte {
		request := string(requestData)

		response := handler(request)

		return []byte(response)
	}
}
