package identity

import (
	"github.com/mysterium/node/service_discovery/dto"
)

type idmFake struct{}

func NewIdentityManagerFake() *idmFake {
	return &idmFake{}
}

func (idmFake *idmFake) Register(id dto.Identity) error {
	return nil
}

func (fakeIdm *idmFake) CreateNewIdentity(_ string) (dto.Identity, error) {
	id := dto.Identity("0x000000000000000000000000000000000000bEEF")
	return id, nil
}
func (fakeIdm *idmFake) GetIdentities() []dto.Identity {
	accountList := []dto.Identity{
		dto.Identity("0x000000000000000000000000000000000000bEEF"),
		dto.Identity("0x000000000000000000000000000000000000bEEF"),
	}

	return accountList
}
func (fakeIdm *idmFake) GetIdentity(string) (dto.Identity, error) {
	id := dto.Identity("0x000000000000000000000000000000000000000A")
	return id, nil
}
func (fakeIdm *idmFake) HasIdentity(string) bool {
	return true
}
