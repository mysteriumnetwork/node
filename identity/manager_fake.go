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

package identity

import "github.com/pkg/errors"

type idmFake struct {
	LastUnlockAddress    string
	LastUnlockPassphrase string
	LastUnlockChainID    int64
	existingIdentities   []Identity
	newIdentity          Identity
	unlockFails          bool
	isUnlocked           bool
}

// NewIdentityManagerFake creates fake identity manager for testing purposes
// TODO each caller should use it's own mocked manager part instead of global one
func NewIdentityManagerFake(existingIdentities []Identity, newIdentity Identity) *idmFake {
	return &idmFake{"", "", 0, existingIdentities, newIdentity, false, true}
}

func (fakeIdm *idmFake) IsUnlocked(id string) bool {
	return fakeIdm.isUnlocked
}

func (fakeIdm *idmFake) MarkUnlockToFail() {
	fakeIdm.unlockFails = true
}

func (fakeIdm *idmFake) CreateNewIdentity(_ string) (Identity, error) {
	return fakeIdm.newIdentity, nil
}
func (fakeIdm *idmFake) GetIdentities() []Identity {
	return fakeIdm.existingIdentities
}

func (fakeIdm *idmFake) GetUnlockedIdentity() (Identity, bool) {
	return fakeIdm.newIdentity, false
}

func (fakeIdm *idmFake) GetIdentity(address string) (Identity, error) {
	for _, fakeIdentity := range fakeIdm.existingIdentities {
		if address == fakeIdentity.Address {
			return fakeIdentity, nil
		}
	}
	return Identity{}, errors.New("Identity not found")
}
func (fakeIdm *idmFake) HasIdentity(_ string) bool {
	return true
}

func (fakeIdm *idmFake) Unlock(chainID int64, address string, passphrase string) error {
	fakeIdm.LastUnlockAddress = address
	fakeIdm.LastUnlockPassphrase = passphrase
	fakeIdm.LastUnlockChainID = chainID
	if fakeIdm.unlockFails {
		return errors.New("Unlock failed")
	}
	return nil
}
