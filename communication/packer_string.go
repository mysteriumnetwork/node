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

func StringListener(callback func(string)) MessageListener {
	return func(data []byte) {
		callback(string(data))
	}
}

func StringHandler(handler func(string) string) RequestHandler {
	return func(data []byte) []byte {
		request := string(data)

		response := handler(request)

		return []byte(response)
	}
}
