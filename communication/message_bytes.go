package communication

func BytesPacker(message []byte) MessagePacker {
	return func() []byte {
		return message
	}
}

func BytesUnpacker(messagePtr *[]byte) MessageUnpacker {
	return func(data []byte) {
		*messagePtr = data
	}
}

func BytesListener(callback func(message []byte)) MessageListener {
	return func(data []byte) {
		callback(data)
	}
}

type BytesResponder struct {
	Callback func(request []byte) []byte
}

func (consumer BytesResponder) ConsumeRequest(requestBody []byte) (responseBody []byte) {
	return consumer.Callback(requestBody)
}
