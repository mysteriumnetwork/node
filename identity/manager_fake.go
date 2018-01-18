package identity

import "github.com/pkg/errors"

type idmFake struct {
	LastUnlockAddress    string
	LastUnlockPassphrase string
	existingIdentities   []Identity
	newIdentity          Identity
	unlockFails          bool
}

func NewIdentityManagerFake(existingIdentities []Identity, newIdentity Identity) *idmFake {
	return &idmFake{"", "", existingIdentities, newIdentity, false}
}

func (fakeIdm *idmFake) MarkUnlockToFail() {
	fakeIdm.unlockFails = true
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
	if fakeIdm.unlockFails {
		return errors.New("Unlock failed")
	}
	return nil
}
