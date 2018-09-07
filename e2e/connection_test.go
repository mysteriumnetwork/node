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
	"github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/stretchr/testify/assert"
)

func TestClientConnectsToNode(t *testing.T) {

	tequilApi := newTequilaClient()

	status, err := tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

	identity, err := tequilApi.NewIdentity("")
	assert.NoError(t, err)
	seelog.Info("Client identity is: ", identity.Address)

	err = tequilApi.Unlock(identity.Address, "")
	assert.NoError(t, err)

	registrationData, err := tequilApi.RegistrationStatus(identity.Address)
	assert.NoError(t, err)

	err = registerIdentity(registrationData)
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		regStatus, err := tequilApi.RegistrationStatus(identity.Address)
		return regStatus.Registered, err
	})
	assert.NoError(t, err)

	nonVpnIp, err := tequilApi.GetIP()
	assert.NoError(t, err)
	seelog.Info("Direct client address is: ", nonVpnIp)

	proposals, err := tequilApi.Proposals()
	if err != nil {
		assert.Error(t, err)
		assert.FailNow(t, "Proposals returned error - no point to continue")
	}

	//expect exactly one proposal
	if len(proposals) != 1 {
		assert.FailNow(t, "Exactly one proposal is expected - something is not right!")
	}

	proposal := proposals[0]
	seelog.Info("Selected proposal is: ", proposal)

	_, err = tequilApi.Connect(identity.Address, proposal.ProviderID, endpoints.ConnectOptions{true})
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilApi.Status()
		return status.Status == "Connected", err
	})
	assert.NoError(t, err)

	vpnIp, err := tequilApi.GetIP()
	assert.NoError(t, err)
	seelog.Info("VPN client address is: ", vpnIp)

	err = tequilApi.Disconnect()
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilApi.Status()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

}
