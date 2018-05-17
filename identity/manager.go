/*
 * Copyright (C) 2018 The Mysterium Network Authors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

// Maps Ethereum account to dto.Identity.
// Currently creates a new eth account with passphrase on CreateNewIdentity().

package identity

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
)

type identityManager struct {
	keystoreManager keystoreInterface
}

// NewIdentityManager creates and returns new identityManager
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
