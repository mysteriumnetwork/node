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

package e2e

import (
	"testing"

	"github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
)

func TestNewClientIdentityRegistrationFlow(t *testing.T) {

	tequilapi := newTequilaClient(Client)

	mystIdentity, err := tequilapi.NewIdentity("")
	assert.NoError(t, err)
	seelog.Info("Created new identity: ", mystIdentity.Address)

	err = tequilapi.Unlock(mystIdentity.Address, "")
	assert.NoError(t, err)

	registrationStatus, err := tequilapi.IdentityRegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.False(t, registrationStatus.Registered)

	err = registerIdentity(registrationStatus)
	assert.NoError(t, err)

	//now we check identity again
	newStatus, err := tequilapi.IdentityRegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.True(t, newStatus.Registered)
}

func TestServerIdentityRegistrationFlow(t *testing.T) {

	tequilapi := newTequilaClient(Server)

	mystIdentity, err := tequilapi.NewIdentity("")
	assert.NoError(t, err)
	seelog.Info("Created new identity: ", mystIdentity.Address)

	err = tequilapi.Unlock(mystIdentity.Address, "")
	assert.NoError(t, err)

	registrationStatus, err := tequilapi.IdentityRegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.False(t, registrationStatus.Registered)

	err = registerIdentity(registrationStatus)
	assert.NoError(t, err)

	newStatus, err := tequilapi.IdentityRegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.False(t, newStatus.Registered)

	t.Run("TestClientConnectsToNode", func(t *testing.T) {
		clientConnectsToNodeTest(t)
	})
}
