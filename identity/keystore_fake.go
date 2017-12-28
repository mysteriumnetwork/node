package identity

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
)

type keyStoreFake struct {
	AccountsMock []accounts.Account
	ErrorMock    error
}

func NewKeystoreFake() *keyStoreFake {
    return &keyStoreFake{}
}

func (self *keyStoreFake) Accounts() []accounts.Account {
	return self.AccountsMock
}

func (self *keyStoreFake) NewAccount(_ string) (accounts.Account, error) {
	if self.ErrorMock != nil {
		return accounts.Account{}, self.ErrorMock
	}

	accountNew := accounts.Account{
		Address: common.HexToAddress("0x000000000000000000000000000000000000bEEF"),
	}
	self.AccountsMock = append(self.AccountsMock, accountNew)

	return accountNew, nil
}

func (self *keyStoreFake) Unlock(a accounts.Account, passphrase string) error {
	if self.ErrorMock != nil {
		return self.ErrorMock
	}

	return nil
}

func (self *keyStoreFake) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	if self.ErrorMock != nil {
		return []byte{}, self.ErrorMock
	}

	panic("implement me")
}

func (self *keyStoreFake) Find(a accounts.Account) (accounts.Account, error) {
	if self.ErrorMock != nil {
		return accounts.Account{}, self.ErrorMock
	}

	panic("implement me")
}

