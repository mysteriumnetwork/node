package identity

import "github.com/ethereum/go-ethereum/accounts"

type keystoreInterface interface {
	Accounts() []accounts.Account
	NewAccount(passphrase string) (accounts.Account, error)
	Find(a accounts.Account) (accounts.Account, error)
	Unlock(a accounts.Account, passphrase string) error
	SignHash(a accounts.Account, hash []byte) ([]byte, error)
}
