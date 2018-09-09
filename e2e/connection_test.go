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
	"github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/stretchr/testify/assert"
)

func TestClientConnectsToNode(t *testing.T) {
	tequilapiClient := newTequilapiClient(Client)

	status, err := tequilapiClient.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

	mystID := identityCreateFlow(t, tequilapiClient)
	identityRegistrationFlow(t, tequilapiClient, mystID)

	nonVpnIp, err := tequilapiClient.GetIP()
	assert.NoError(t, err)
	seelog.Info("Original client IP: ", nonVpnIp)

	proposals, err := tequilapiClient.Proposals()
	if err != nil {
		assert.Error(t, err)
		assert.FailNow(t, "Proposals returned error - no point to continue")
	}

	// expect exactly one proposal
	if len(proposals) != 1 {
		assert.FailNow(t, "Exactly one proposal is expected - something is not right!")
	}

	proposal := proposals[0]
	seelog.Info("Selected proposal is: ", proposal)

	_, err = tequilapiClient.Connect(mystID.Address, proposal.ProviderID)
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapiClient.Status()
		return status.Status == "Connected", err
	})
	assert.NoError(t, err)

	vpnIp, err := tequilapiClient.GetIP()
	assert.NoError(t, err)
	seelog.Info("Shifted client IP: ", vpnIp)

	err = tequilapiClient.Disconnect()
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapiClient.Status()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

}

func identityCreateFlow(t *testing.T, tequilapi *client.Client) client.IdentityDTO {
	id, err := tequilapi.NewIdentity("")
	assert.NoError(t, err)
	seelog.Info("Created new identity: ", id.Address)

	err = tequilapi.Unlock(id.Address, "")
	assert.NoError(t, err)

	return id
}

func identityRegistrationFlow(t *testing.T, tequilapi *client.Client, id client.IdentityDTO) {
	registrationData, err := tequilapi.IdentityRegistrationStatus(id.Address)
	assert.NoError(t, err)
	assert.False(t, registrationData.Registered)

	err = registerIdentity(registrationData)
	assert.NoError(t, err)
	seelog.Info("Registered identity: ", id.Address)

	// now we check identity again
	err = waitForCondition(func() (bool, error) {
		regStatus, err := tequilapi.IdentityRegistrationStatus(id.Address)
		return regStatus.Registered, err
	})
	assert.NoError(t, err)
}
