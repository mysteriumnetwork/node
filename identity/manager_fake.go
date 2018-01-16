package identity

import "github.com/pkg/errors"

type idmFake struct {
	LastUnlockAddress    string
	LastUnlockPassphrase string
	UnlockFails          bool
	existingIdentities   []Identity
	newIdentity          Identity
}

func NewIdentityManagerFake(existingIdentities []Identity, newIdentity Identity) *idmFake {
	return &idmFake{"", "", false, existingIdentities, newIdentity}
}

func (fakeIdm *idmFake) CreateNewIdentity(_ string) (Identity, error) {
	return fakeIdm.newIdentity, nil
}
func (fakeIdm *idmFake) GetIdentities() []Identity {
	return fakeIdm.existingIdentities
}
func (fakeIdm *idmFake) GetIdentity(address string) (Identity, error) {
	for _, fakeIdentity := range fakeIdm.existingIdentities {
		if address == fakeIdentity.Address {
			return fakeIdentity, nil
		}
	}
	return Identity{}, errors.New("Identity not found")
}
func (fakeIdm *idmFake) HasIdentity(_ string) bool {
	return true
}

func (fakeIdm *idmFake) Unlock(address string, passphrase string) error {
	fakeIdm.LastUnlockAddress = address
	fakeIdm.LastUnlockPassphrase = passphrase
	if fakeIdm.UnlockFails {
		return errors.New("Unlock failed")
	}
	return nil
}

func (fakeIdm *idmFake) CleanStatus() {
	fakeIdm.LastUnlockAddress = ""
	fakeIdm.LastUnlockPassphrase = ""
	fakeIdm.UnlockFails = false
}
