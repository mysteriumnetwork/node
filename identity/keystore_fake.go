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

func (self *keyStoreFake) NewAccount(address string) (accounts.Account, error) {
	if self.ErrorMock != nil {
		return accounts.Account{}, self.ErrorMock
	}

	accountNew := accounts.Account{
		Address: common.HexToAddress(address),
	}
	self.AccountsMock = append(self.AccountsMock, accountNew)

	return accountNew, nil
}

func (self *keyStoreFake) Unlock(a accounts.Account, passphrase string) error {
	if self.ErrorMock != nil {
		return self.ErrorMock
	}

	panic("implement me")
}

func (self *keyStoreFake) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	if self.ErrorMock != nil {
		return []byte{}, self.ErrorMock
	}

	panic("implement me")
}
