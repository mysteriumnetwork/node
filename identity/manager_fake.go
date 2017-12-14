package identity

import (
	"github.com/mysterium/node/service_discovery/dto"
)

type IdentityManagerFake struct {
}

func NewIdentityManagerFake() *IdentityManagerFake {
	return &IdentityManagerFake{}
}

func (fakeIdm *IdentityManagerFake) CreateNewIdentity(_ string) (*dto.Identity, error) {
	id := dto.Identity("0x000000000000000000000000000000000000000A")
	return &id, nil
}
func (fakeIdm *IdentityManagerFake) GetIdentities() []dto.Identity {
	return []dto.Identity{}
}
func (fakeIdm *IdentityManagerFake) GetIdentity(string) *dto.Identity {
	id := dto.Identity("0x000000000000000000000000000000000000000A")
	return &id
}
func (fakeIdm *IdentityManagerFake) HasIdentity(string) bool {
	return true
}
