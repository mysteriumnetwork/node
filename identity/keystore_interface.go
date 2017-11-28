package identity

import "github.com/ethereum/go-ethereum/accounts"

type keystoreManager interface {
	Accounts() []accounts.Account
	NewAccount(passphrase string) (accounts.Account, error)
}
