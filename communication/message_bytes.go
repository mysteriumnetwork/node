package communication

type BytesProduce struct {
	Message []byte
}

func (producer BytesProduce) ProduceMessage() []byte {
	return producer.Message
}

type BytesCallback struct {
	Callback func(message []byte)
}

func (consumer BytesCallback) ConsumeMessage(messageBody []byte) {
	consumer.Callback(messageBody)
}
