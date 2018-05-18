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

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newManager(accountValue string) *identityManager {
	return &identityManager{
		keystoreManager: &keyStoreFake{
			AccountsMock: []accounts.Account{
				addressToAccount(accountValue),
			},
		},
	}
}

func newManagerWithError(errorMock error) *identityManager {
	return &identityManager{
		keystoreManager: &keyStoreFake{
			ErrorMock: errorMock,
		},
	}
}

func TestManager_CreateNewIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")
	identity, err := manager.CreateNewIdentity("")

	assert.NoError(t, err)
	assert.Equal(t, identity, Identity{"0x000000000000000000000000000000000000beef"})
	assert.Len(t, manager.keystoreManager.Accounts(), 2)
}

func TestManager_CreateNewIdentityError(t *testing.T) {
	im := newManagerWithError(errors.New("identity create failed"))
	identity, err := im.CreateNewIdentity("")

	assert.EqualError(t, err, "identity create failed")
	assert.Empty(t, identity.Address)
}

func TestManager_GetIdentities(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.Equal(
		t,
		[]Identity{
			{"0x000000000000000000000000000000000000000a"},
		},
		manager.GetIdentities(),
	)
}

func TestManager_GetIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	identity, err := manager.GetIdentity("0x000000000000000000000000000000000000000A")
	assert.NoError(t, err)
	assert.Exactly(t, Identity{"0x000000000000000000000000000000000000000a"}, identity)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000a")
	assert.NoError(t, err)
	assert.Exactly(t, Identity{"0x000000000000000000000000000000000000000a"}, identity)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000B")
	assert.EqualError(t, err, "identity not found")
	assert.Exactly(t, Identity{}, identity)
}

func TestManager_HasIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000A"))
	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000a"))
	assert.False(t, manager.HasIdentity("0x000000000000000000000000000000000000000B"))
}
