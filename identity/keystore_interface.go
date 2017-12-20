package identity

import "github.com/ethereum/go-ethereum/accounts"

type keystoreInterface interface {
	Accounts() []accounts.Account
	NewAccount(passphrase string) (accounts.Account, error)
}
