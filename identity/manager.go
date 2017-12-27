// Maps Ethereum account to dto.Identity.
// Currently creates a new eth account with password on CreateNewIdentity().

package identity

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"strings"
	"github.com/ethereum/go-ethereum/accounts/keystore"
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
	account := accounts.Account{
		Address: common.HexToAddress(identity.Address),
	}

	return account
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

func (idm *identityManager) Unlock(identity Identity) error {
	keystoreManager := idm.keystoreManager
	accountExisting := identityToAccount(identity)

	account, err := keystoreManager.Find(accountExisting)
	if err != nil {
		return err
	}

	err = keystoreManager.Unlock(account, "")
	if err != nil {
		return err
	}

	return nil
}

func (idm *identityManager) IsUnlocked(identity Identity) bool {
	signer := idm.GetSigner(identity)

	_, err := signer.Sign([]byte("1"))
	if err == keystore.ErrLocked {
		return false
	}

	return true
}

func (idm *identityManager) GetSigner(identity Identity) Signer {
	return NewSigner(idm.keystoreManager, identity)
}

func (idm *identityManager) GetIdentity(identityString string) (identity Identity, err error) {
	identityString = strings.ToLower(identityString)
	for _, identity := range idm.GetIdentities() {
		if strings.ToLower(identity.Address) == identityString {
			return identity, nil
		}
	}

	return identity, errors.New("identity not found")
}

func (idm *identityManager) HasIdentity(identityString string) bool {
	_, err := idm.GetIdentity(identityString)

	return err == nil
}
