package identity

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"
)

const PASSPHRASE = ""

type IdentityManager struct {
	keystoreManager *keystore.KeyStore
}

func NewIdentityManager(keydir string) *IdentityManager {
	return &IdentityManager{
		keystoreManager: keystore.NewKeyStore(keydir, keystore.StandardScryptN, keystore.StandardScryptP),
	}
}

func (idm *IdentityManager) CreateNewIdentity() (string, error) {
	account, err := idm.keystoreManager.NewAccount(PASSPHRASE)
	if err != nil {
		return "", err
	}

	return account.Address.Hex(), nil
}

func (idm *IdentityManager) GetIdentities() []string {
	var ids []string
	for _, account := range idm.keystoreManager.Accounts() {
		ids = append(ids, account.Address.Hex())
	}
	return ids
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

func (idm *IdentityManager) SignMessage(identity string, message string) ([]byte, error) {
	accountExisting := accounts.Account{
		Address: common.HexToAddress(identity),
	}

	account, err := idm.keystoreManager.Find(accountExisting)
	if err != nil {
		return nil, err
	}

	err =  idm.keystoreManager.Unlock(account, PASSPHRASE)
	if err != nil {
		return nil, err
	}

	signature, err :=  idm.keystoreManager.SignHash(account, signHash([]byte(message)))
	if err != nil {
		return nil, err
	}

	return signature, nil
}
