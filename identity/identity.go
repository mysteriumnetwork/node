package identity

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"
)

func CreateNewIdentity(path string) (string, error) {
	keystoreManager := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := keystoreManager.NewAccount("")
	if err != nil {
		return "", err
	}

	return account.Address.Hex(), nil
}

func GetIdentities(path string) []string {
	keystoreManager := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	var ids []string
	for _, account := range keystoreManager.Accounts() {
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

func SignMessage(path string, identity string, message string) ([]byte, error) {
	keystoreManager := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	accountExisting := accounts.Account{
		Address: common.HexToAddress(identity),
	}

	account, err := keystoreManager.Find(accountExisting)
	if err != nil {
		return nil, err
	}

	err = keystoreManager.Unlock(account, "")
	if err != nil {
		return nil, err
	}

	signature, err := keystoreManager.SignHash(account, signHash([]byte(message)))
	if err != nil {
		return nil, err
	}

	return signature, nil
}