package session

type GeneratorFake struct {
	SessionIdMock SessionId
}

func (generator *GeneratorFake) Generate() SessionId {
	return generator.SessionIdMock
}
