package communication

type StringProduce struct {
	Message string
}

func (producer StringProduce) ProduceMessage() []byte {
	return []byte(producer.Message)
}

type StringResponse struct {
	Response string
}

func (consumer *StringResponse) ConsumeMessage(messageBody []byte) {
	consumer.Response = string(messageBody)
}

type StringCallback struct {
	Callback func(message string)
}

func (consumer StringCallback) ConsumeMessage(messageBody []byte) {
	consumer.Callback(string(messageBody))
}
