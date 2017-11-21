package identity

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
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