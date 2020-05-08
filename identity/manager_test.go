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

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/eventbus"
)

func Test_IdentityManager(t *testing.T) {
	ks := NewMockKeystoreWith(MockKeys)
	idm := &identityManager{
		keystoreManager: ks,
		eventBus:        eventbus.New(),
		unlocked:        map[string]bool{},
	}

	t.Run("gets existing identities", func(t *testing.T) {
		identities := idm.GetIdentities()
		assert.Len(t, identities, 1)
		assert.Equal(t, FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"), identities[0])
	})

	var newID Identity
	t.Run("creates a new identity", func(t *testing.T) {
		id, err := idm.CreateNewIdentity("")
		assert.NoError(t, err)
		assert.Len(t, idm.keystoreManager.Accounts(), 2)
		assert.True(t, common.IsHexAddress(id.Address))
		newID = id
	})

	t.Run("gets identity", func(t *testing.T) {
		identity, err := idm.GetIdentity("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68")
		assert.NoError(t, err)
		assert.Exactly(t, Identity{"0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"}, identity)

		identity, err = idm.GetIdentity(newID.Address)
		assert.NoError(t, err)
		assert.Exactly(t, newID, identity)

		identity, err = idm.GetIdentity("0x000000000000000000000000000000000000000B")
		assert.EqualError(t, err, "identity not found")
		assert.Exactly(t, Identity{}, identity)
	})

	t.Run("has identity", func(t *testing.T) {
		assert.True(t, idm.HasIdentity("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
		assert.True(t, idm.HasIdentity(newID.Address))
		assert.False(t, idm.HasIdentity("0x000000000000000000000000000000000000000B"))
	})
}
