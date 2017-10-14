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

type BytesResponder struct {
	Callback func(request []byte) []byte
}

func (consumer BytesResponder) ConsumeRequest(requestBody []byte) (responseBody []byte) {
	return consumer.Callback(requestBody)
}
