package communication

func BytesPacker(message []byte) Packer {
	return func() []byte {
		return message
	}
}

func BytesUnpacker(messagePtr *[]byte) Unpacker {
	return func(data []byte) {
		*messagePtr = data
	}
}

func BytesListener(callback func([]byte)) MessageListener {
	return callback
}

func BytesHandler(callback func([]byte) []byte) RequestHandler {
	return callback
}
