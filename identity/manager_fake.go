package identity

import "github.com/pkg/errors"

type idmFake struct {
	existingIdentities []Identity
	newIdentity        Identity
}

func NewIdentityManagerFake(existingIdentities []Identity, newIdentity Identity) *idmFake {
	return &idmFake{existingIdentities, newIdentity}
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
