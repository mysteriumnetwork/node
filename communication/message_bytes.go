package communication

type BytesProduce struct {
	Message []byte
}

func (producer BytesProduce) ProduceMessage() []byte {
	return producer.Message
}

type BytesResponse struct {
	Response []byte
}

func (consumer *BytesResponse) ConsumeMessage(messageBody []byte) {
	consumer.Response = messageBody
}

type BytesCallback struct {
	Callback func(message []byte)
}

func (consumer BytesCallback) ConsumeMessage(messageBody []byte) {
	consumer.Callback(messageBody)
}

type BytesResponder struct {
	Callback func(request []byte) []byte
}

func (consumer BytesResponder) ConsumeRequest(requestBody []byte) (responseBody []byte) {
	return consumer.Callback(requestBody)
}
