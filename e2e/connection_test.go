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
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/stretchr/testify/assert"
)

var (
	consumerPassphrase = "localconsumer"
	providerID         = "0xd1a23227bd5ad77f36ba62badcb78a410a1db6c5"
	providerPassphrase = "localprovider"
)

func TestConsumerConnectsToProvider(t *testing.T) {
	tequilapiProvider := newTequilapiProvider()
	tequilapiConsumer := newTequilapiConsumer()

	t.Run("ProviderRegistersIdentityFlow", func(t *testing.T) {
		identityRegistrationFlow(t, tequilapiProvider, providerID, providerPassphrase)
	})

	var consumerID string
	t.Run("ConsumerCreatesAndRegistersIdentityFlow", func(t *testing.T) {
		consumerID = identityCreateFlow(t, tequilapiConsumer, consumerPassphrase)
		identityRegistrationFlow(t, tequilapiConsumer, consumerID, consumerPassphrase)
	})

	t.Run("ConsumerConnectFlow", func(t *testing.T) {
		proposal := consumerPicksProposal(t, tequilapiConsumer)
		consumerConnectFlow(t, tequilapiConsumer, consumerID, proposal)
	})
}

func identityCreateFlow(t *testing.T, tequilapi *tequilapi_client.Client, idPassphrase string) string {
	id, err := tequilapi.NewIdentity(idPassphrase)
	assert.NoError(t, err)
	seelog.Info("Created new identity: ", id.Address)

	return id.Address
}

func identityRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
	err := tequilapi.Unlock(id, idPassphrase)
	assert.NoError(t, err)

	registrationData, err := tequilapi.IdentityRegistrationStatus(id)
	assert.NoError(t, err)
	assert.False(t, registrationData.Registered)

	err = registerIdentity(registrationData)
	assert.NoError(t, err)
	seelog.Info("Registered identity: ", id)

	// now we check identity again
	err = waitForCondition(func() (bool, error) {
		regStatus, err := tequilapi.IdentityRegistrationStatus(id)
		return regStatus.Registered, err
	})
	assert.NoError(t, err)
}

// expect exactly one proposal
func consumerPicksProposal(t *testing.T, tequilapi *tequilapi_client.Client) tequilapi_client.ProposalDTO {
	var proposals []tequilapi_client.ProposalDTO
	err := waitForCondition(func() (state bool, stateErr error) {
		proposals, stateErr = tequilapi.Proposals()
		return len(proposals) == 1, stateErr
	})
	if err != nil {
		assert.Error(t, err)
		assert.FailNow(t, "Exactly one proposal is expected - something is not right!")
	}

	seelog.Info("Selected proposal is: ", proposals[0])
	return proposals[0]
}

func consumerConnectFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID string, proposal tequilapi_client.ProposalDTO) {
	status, err := tequilapi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

	nonVpnIp, err := tequilapi.GetIP()
	assert.NoError(t, err)
	seelog.Info("Original consumer IP: ", nonVpnIp)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.Status()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	_, err = tequilapi.Connect(consumerID, proposal.ProviderID)
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.Status()
		return status.Status == "Connected", err
	})
	assert.NoError(t, err)

	vpnIp, err := tequilapi.GetIP()
	assert.NoError(t, err)
	seelog.Info("Shifted consumer IP: ", vpnIp)

	err = tequilapi.Disconnect()
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.Status()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)
}
