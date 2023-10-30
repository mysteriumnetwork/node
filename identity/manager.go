/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/eventbus"
)

// Identity events
const (
	AppTopicIdentityUnlock  = "identity-unlocked"
	AppTopicIdentityCreated = "identity-created"
)

// AppEventIdentityUnlock represents the payload that is sent on identity unlock.
type AppEventIdentityUnlock struct {
	ChainID int64
	ID      Identity
}

// ResidentCountryEvent represent actual resident country changed event
type ResidentCountryEvent struct {
	ID      string
	Country string
}

type identityManager struct {
	keystoreManager keystore
	residentCountry *ResidentCountry
	unlocked        map[string]bool // Currently unlocked addresses
	unlockedMu      sync.RWMutex
	eventBus        eventbus.EventBus
}

// keystore allows actions with accounts (listing, creating, unlocking, signing)
type keystore interface {
	Accounts() []accounts.Account
	NewAccount(passphrase string) (accounts.Account, error)
	Find(a accounts.Account) (accounts.Account, error)
	Unlock(a accounts.Account, passphrase string) error
	SignHash(a accounts.Account, hash []byte) ([]byte, error)
}

// NewIdentityManager creates and returns new identityManager
func NewIdentityManager(keystore keystore, eventBus eventbus.EventBus, residentCountry *ResidentCountry) *identityManager {
	return &identityManager{
		keystoreManager: keystore,
		residentCountry: residentCountry,
		unlocked:        map[string]bool{},
		eventBus:        eventBus,
	}
}

// GetUnlockedIdentity retrieves unlocked identity
func (idm *identityManager) GetUnlockedIdentity() (Identity, bool) {
	for _, identity := range idm.GetIdentities() {
		if idm.IsUnlocked(identity.Address) {
			return identity, true
		}
	}
	return Identity{}, false
}

// IsUnlocked checks if the given identity is unlocked or not
func (idm *identityManager) IsUnlocked(identity string) bool {
	idm.unlockedMu.Lock()
	defer idm.unlockedMu.Unlock()
	_, ok := idm.unlocked[identity]
	return ok
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

	identity = accountToIdentity(account)
	idm.eventBus.Publish(AppTopicIdentityCreated, identity.Address)
	return identity, nil
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

func (idm *identityManager) Unlock(chainID int64, address string, passphrase string) error {
	idm.unlockedMu.Lock()
	defer idm.unlockedMu.Unlock()

	if idm.unlocked[address] {
		log.Debug().Msg("Unlocked identity found in cache, skipping keystore: " + address)
		return nil
	}

	account, err := idm.findAccount(address)
	if err != nil {
		return err
	}

	err = idm.keystoreManager.Unlock(account, passphrase)
	if err != nil {
		return errors.Wrapf(err, "keystore failed to unlock identity: %s", address)
	}
	log.Debug().Msgf("Caching unlocked address: %s", address)
	idm.unlocked[address] = true

	go func() {
		idm.eventBus.Publish(AppTopicIdentityUnlock, AppEventIdentityUnlock{
			ChainID: chainID,
			ID:      FromAddress(address),
		})
		idm.residentCountry.publishResidentCountry(address)
	}()

	return nil
}

func (idm *identityManager) findAccount(address string) (accounts.Account, error) {
	account, err := idm.keystoreManager.Find(addressToAccount(address))
	if err != nil {
		return accounts.Account{}, errors.New("identity not found: " + address)
	}

	return account, err
}
