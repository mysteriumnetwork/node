// Maps Ethereum account to dto.Identity.
// Currently creates a new eth account with password on CreateNewIdentity().

package identity

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type identityManager struct {
	keystoreManager keystoreInterface
}

func NewIdentityManager(keystore keystoreInterface) *identityManager {
	return &identityManager{
		keystoreManager: keystore,
	}
}

func accountToIdentity(account accounts.Account) Identity {
	identity := FromAddress(account.Address.Hex())
	return identity
}

func identityToAccount(identity Identity) accounts.Account {
	return addressToAccount(identity.Address)
}

func addressToAccount(identityAddress string) accounts.Account {
	return accounts.Account{
		Address: common.HexToAddress(identityAddress),
	}
}

func (idm *identityManager) CreateNewIdentity(passphrase string) (identity Identity, err error) {
	account, err := idm.keystoreManager.NewAccount(passphrase)
	if err != nil {
		return identity, err
	}

	return accountToIdentity(account), nil
}

func (idm *identityManager) GetIdentities() []Identity {
	accountList := idm.keystoreManager.Accounts()

	var ids = make([]Identity, len(accountList))
	for i, account := range accountList {
		ids[i] = accountToIdentity(account)
	}

	return ids
}

func (idm *identityManager) GetIdentity(identityString string) (identity Identity, err error) {
	account, err := idm.findAccount(identityString)
	if err != nil {
		return identity, errors.New("identity not found")
	}

	return accountToIdentity(account), nil
}

func (idm *identityManager) HasIdentity(identityString string) bool {
	_, err := idm.findAccount(identityString)
	return err == nil
}

func (idm *identityManager) Unlock(identityString string, passphrase string) error {
	account, err := idm.findAccount(identityString)
	if err != nil {
		return err
	}

	return idm.keystoreManager.Unlock(account, passphrase)
}

func (idm *identityManager) findAccount(identityString string) (accounts.Account, error) {
	account, err := idm.keystoreManager.Find(addressToAccount(identityString))
	if err != nil {
		return accounts.Account{}, errors.New("identity not found: " + identityString)
	}

	return account, err
}
