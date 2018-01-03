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

func addressToAccount(address string) accounts.Account {
	return accounts.Account{
		Address: common.HexToAddress(address),
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

func (idm *identityManager) GetIdentity(address string) (identity Identity, err error) {
	account, err := idm.findAccount(address)
	if err != nil {
		return identity, errors.New("identity not found")
	}

	return accountToIdentity(account), nil
}

func (idm *identityManager) HasIdentity(address string) bool {
	_, err := idm.findAccount(address)
	return err == nil
}

func (idm *identityManager) Unlock(address string, passphrase string) error {
	account, err := idm.findAccount(address)
	if err != nil {
		return err
	}

	return idm.keystoreManager.Unlock(account, passphrase)
}

func (idm *identityManager) findAccount(address string) (accounts.Account, error) {
	account, err := idm.keystoreManager.Find(addressToAccount(address))
	if err != nil {
		return accounts.Account{}, errors.New("identity not found: " + address)
	}

	return account, err
}
