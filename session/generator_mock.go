package session

type GeneratorMock struct{}

func (generator *GeneratorMock) Generate() SessionId {
	return SessionId("")
}
