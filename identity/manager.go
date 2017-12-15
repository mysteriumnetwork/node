// Maps Ethereum account to dto.Identity.
// Currently creates a new eth account with password on CreateNewIdentity().

package identity

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysterium/node/service_discovery/dto"
	"strings"
)

type identityManager struct {
	keystoreManager keystoreInterface
}

func NewIdentityManager(keystore keystoreInterface) *identityManager {
	return &identityManager{
		keystoreManager: keystore,
	}
}

func accountToIdentity(account accounts.Account) dto.Identity {
	identity := dto.Identity(account.Address.Hex())
	return identity
}

func identityToAccount(identityStr string) accounts.Account {
	return accounts.Account{
		Address: common.HexToAddress(identityStr),
	}
}

func (idm *identityManager) CreateNewIdentity(passphrase string) (dto.Identity, error) {
	account, err := idm.keystoreManager.NewAccount(passphrase)
	if err != nil {
		return "", err
	}

	return accountToIdentity(account), nil
}

func (idm *identityManager) GetIdentities() []dto.Identity {
	accountList := idm.keystoreManager.Accounts()

	var ids = make([]dto.Identity, len(accountList))
	for i, account := range accountList {
		ids[i] = accountToIdentity(account)
	}

	return ids
}

func (idm *identityManager) GetIdentity(identityString string) dto.Identity {
	identityString = strings.ToLower(identityString)
	for _, id := range idm.GetIdentities() {
		if strings.ToLower(string(id)) == identityString {
			return id
		}
	}

	return dto.Identity("")
}

func (idm *identityManager) HasIdentity(identityString string) bool {
	return len(idm.GetIdentity(identityString)) != 0
}
