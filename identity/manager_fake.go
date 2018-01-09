package identity

type idmFake struct{
	FakeIdentity1 Identity
	FakeIdentity2 Identity
}

func NewIdentityManagerFake() *idmFake {
	return &idmFake{
		FromAddress("0x000000000000000000000000000000000000000A"),
		FromAddress("0x000000000000000000000000000000000000bEEF"),
	}
}

func (fakeIdm *idmFake) CreateNewIdentity(_ string) (Identity, error) {
	return fakeIdm.FakeIdentity2, nil
}
func (fakeIdm *idmFake) GetIdentities() []Identity {
	accountList := []Identity{
		fakeIdm.FakeIdentity2,
		fakeIdm.FakeIdentity2,
	}

	return accountList
}
func (fakeIdm *idmFake) GetIdentity(_ string) (Identity, error) {
	return fakeIdm.FakeIdentity1, nil
}
func (fakeIdm *idmFake) HasIdentity(_ string) bool {
	return true
}
