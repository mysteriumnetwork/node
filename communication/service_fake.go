package communication

type serviceFake struct {
}

func (service *serviceFake) Start() error {
	return nil
}

func (service *serviceFake) Stop() error {
	return nil
}

func (service *serviceFake) Send(messageType MessageType, messagePayload string) error {
	return nil
}
