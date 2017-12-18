package identity

type idmFake struct{}

func NewIdentityManagerFake() *idmFake {
	return &idmFake{}
}

func (fakeIdm *idmFake) CreateNewIdentity(_ string) (Identity, error) {
	id := NewIdentity("0x000000000000000000000000000000000000bEEF")
	return id, nil
}
func (fakeIdm *idmFake) GetIdentities() []Identity {
	accountList := []Identity{
		NewIdentity("0x000000000000000000000000000000000000bEEF"),
		NewIdentity("0x000000000000000000000000000000000000bEEF"),
	}

	return accountList
}
func (fakeIdm *idmFake) GetIdentity(string) (Identity, error) {
	id := NewIdentity("0x000000000000000000000000000000000000000A")
	return id, nil
}
func (fakeIdm *idmFake) HasIdentity(string) bool {
	return true
}
