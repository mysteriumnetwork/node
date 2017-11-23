package identity

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"
	"github.com/mysterium/node/service_discovery/dto"
	"strings"
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

func accountToIdentity(account accounts.Account) dto.Identity {
	return dto.Identity(account.Address.Hex())
}

func identityToAccount(identityString string) accounts.Account {
	return accounts.Account{
		Address: common.HexToAddress(identityString),
	}
}

func (idm *IdentityManager) CreateNewIdentity() (dto.Identity, error) {
	account, err := idm.keystoreManager.NewAccount(PASSPHRASE)
	if err != nil {
		return "", err
	}

	return accountToIdentity(account), nil
}

func (idm *IdentityManager) GetIdentities() []dto.Identity {
	var ids []dto.Identity
	for _, account := range idm.keystoreManager.Accounts() {
		ids = append(ids, accountToIdentity(account))
	}

	return ids
}

func (idm *IdentityManager) GetIdentity(identityString string) *dto.Identity {
	identityString = strings.ToLower(identityString)
	for _, id := range idm.GetIdentities() {
		if strings.ToLower(string(id)) == identityString {
			return &id
		}
	}

	return nil
}

func (idm *IdentityManager) HasIdentity(identityString string) bool {
	return idm.GetIdentity(identityString)!= nil
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

func (idm *IdentityManager) SignMessage(identity dto.Identity, message string) ([]byte, error) {
	account := identityToAccount(string(identity))
	err :=  idm.keystoreManager.Unlock(account, PASSPHRASE)
	if err != nil {
		return nil, err
	}

	signature, err :=  idm.keystoreManager.SignHash(account, signHash([]byte(message)))
	if err != nil {
		return nil, err
	}

	return signature, nil
}
