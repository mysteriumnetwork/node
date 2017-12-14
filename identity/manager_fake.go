package identity

import (
	"github.com/mysterium/node/service_discovery/dto"
)

type idmFake struct {
    ksmFake keystoreInterface
}

func NewIdentityManagerFake() *idmFake {
	return &idmFake{
	    ksmFake:NewKeystoreFake(),
    }
}

func (fakeIdm *idmFake) CreateNewIdentity(hexAddress string) (*dto.Identity, error) {
    id, error := fakeIdm.ksmFake.NewAccount(hexAddress)
	//id := dto.Identity("0x000000000000000000000000000000000000000A")
	return accountToIdentity(id), error
}
func (fakeIdm *idmFake) GetIdentities() []dto.Identity {
    accountList := fakeIdm.ksmFake.Accounts()

    var ids = make([]dto.Identity, len(accountList))
    for i, account := range accountList {
        ids[i] = *accountToIdentity(account)
    }

    return ids
}
func (fakeIdm *idmFake) GetIdentity(string) *dto.Identity {
	id := dto.Identity("0x000000000000000000000000000000000000000A")
	return &id
}
func (fakeIdm *idmFake) HasIdentity(string) bool {
	return true
}
