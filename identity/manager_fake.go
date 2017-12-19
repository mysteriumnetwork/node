package identity

type idmFake struct{}

func NewIdentityManagerFake() *idmFake {
	return &idmFake{}
}

func (fakeIdm *idmFake) CreateNewIdentity(_ string) (Identity, error) {
	id := FromAddress("0x000000000000000000000000000000000000bEEF")
	return id, nil
}
func (fakeIdm *idmFake) GetIdentities() []Identity {
	accountList := []Identity{
		FromAddress("0x000000000000000000000000000000000000bEEF"),
		FromAddress("0x000000000000000000000000000000000000bEEF"),
	}

	return accountList
}
func (fakeIdm *idmFake) GetIdentity(string) (Identity, error) {
	id := FromAddress("0x000000000000000000000000000000000000000A")
	return id, nil
}
func (fakeIdm *idmFake) HasIdentity(string) bool {
	return true
}
