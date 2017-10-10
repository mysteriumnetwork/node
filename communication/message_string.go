package communication

type StringProduce struct {
	Message string
}

func (producer StringProduce) ProduceMessage() []byte {
	return []byte(producer.Message)
}

type StringCallback struct {
	Callback func(message string)
}

func (consumer StringCallback) ConsumeMessage(messageBody []byte) {
	consumer.Callback(string(messageBody))
}
