/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

import "errors"

type idmFake struct {
	LastUnlockAddress    string
	LastUnlockPassphrase string
	existingIdentities   []Identity
	newIdentity          Identity
	unlockFails          bool
}

func NewIdentityManagerFake(existingIdentities []Identity, newIdentity Identity) *idmFake {
	return &idmFake{"", "", existingIdentities, newIdentity, false}
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

func (fakeIdm *idmFake) Unlock(address string, passphrase string) error {
	fakeIdm.LastUnlockAddress = address
	fakeIdm.LastUnlockPassphrase = passphrase
	if fakeIdm.unlockFails {
		return errors.New("Unlock failed")
	}
	return nil
}
