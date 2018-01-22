package session

type GeneratorFake struct {
	SessionIdMock SessionID
}

func (generator *GeneratorFake) Generate() SessionID {
	return generator.SessionIdMock
}
