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

package e2e

import (
	"testing"

	"github.com/cihub/seelog"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/endpoints"
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
		for serviceType := range serviceTypeAssertionMap {
			t.Run(serviceType, func(t *testing.T) {
				proposal := consumerPicksProposal(t, tequilapiConsumer, serviceType)
				consumerConnectFlow(t, tequilapiConsumer, consumerID, serviceType, proposal)
			})
		}
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
func consumerPicksProposal(t *testing.T, tequilapi *tequilapi_client.Client, serviceType string) tequilapi_client.ProposalDTO {
	var proposals []tequilapi_client.ProposalDTO
	err := waitForCondition(func() (state bool, stateErr error) {
		proposals, stateErr = tequilapi.ProposalsByType(serviceType)
		return len(proposals) == 1, stateErr
	})
	if err != nil {
		assert.FailNowf(t, "Exactly one proposal is expected - something is not right!", "Error was: %v", err)
	}

	seelog.Info("Selected proposal is: ", proposals[0], ", ServiceType:", serviceType)
	return proposals[0]
}

// filterSessionsByType removes all sessions of irrelevant types
func filterSessionsByType(serviceType string, sessions endpoints.SessionsDTO) endpoints.SessionsDTO {
	matches := 0
	for _, s := range sessions.Sessions {
		if s.ServiceType == serviceType {
			sessions.Sessions[matches] = s
			matches++
		}
	}
	sessions.Sessions = sessions.Sessions[:matches]
	return sessions
}

func getSessionsByType(tequilapi *tequilapi_client.Client, serviceType string) (endpoints.SessionsDTO, error) {
	sessionsDTO, err := tequilapi.GetSessions()
	if err != nil {
		return sessionsDTO, err
	}
	sessionsDTO = filterSessionsByType(serviceType, sessionsDTO)
	return sessionsDTO, nil
}

func consumerConnectFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID, serviceType string, proposal tequilapi_client.ProposalDTO) {
	err := topUpAccount(consumerID)
	assert.Nil(t, err)

	connectionStatus, err := tequilapi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", connectionStatus.Status)

	nonVpnIP, err := tequilapi.GetIP()
	assert.NoError(t, err)
	seelog.Info("Original consumer IP: ", nonVpnIP)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.Status()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	connectionStatus, err = tequilapi.Connect(consumerID, proposal.ProviderID, serviceType, endpoints.ConnectOptions{
		DisableKillSwitch: true,
	})

	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.Status()
		return status.Status == "Connected", err
	})
	assert.NoError(t, err)

	vpnIP, err := tequilapi.GetIP()
	assert.NoError(t, err)
	seelog.Info("Changed consumer IP: ", vpnIP)

	// sessions history should be created after connect
	sessionsDTO, err := getSessionsByType(tequilapi, serviceType)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sessionsDTO.Sessions))
	se := sessionsDTO.Sessions[0]
	assert.Equal(t, uint64(0), se.Duration)
	assert.Equal(t, uint64(0), se.BytesSent)
	assert.Equal(t, uint64(0), se.BytesReceived)
	assert.Equal(t, "e2e-land", se.ProviderCountry)
	assert.Equal(t, serviceType, se.ServiceType)
	assert.Equal(t, proposal.ProviderID, se.ProviderID)
	assert.Equal(t, connectionStatus.SessionID, se.SessionID)
	assert.Equal(t, "New", se.Status)

	err = tequilapi.Disconnect()
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.Status()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	// sessions history should be updated after disconnect
	sessionsDTO, err = getSessionsByType(tequilapi, serviceType)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sessionsDTO.Sessions))
	se = sessionsDTO.Sessions[0]

	// call the custom asserter for the given service type
	serviceTypeAssertionMap[serviceType](t, se)
}

type sessionAsserter func(t *testing.T, session endpoints.SessionDTO)

var serviceTypeAssertionMap = map[string]sessionAsserter{
	"openvpn":   assertOpenvpn,
	"noop":      assertNoop,
	"wireguard": assertWireguard,
}

func assertOpenvpn(t *testing.T, session endpoints.SessionDTO) {
	assert.NotEqual(t, uint64(0), session.BytesSent)
	assert.NotEqual(t, uint64(0), session.BytesReceived)
	assert.Equal(t, "Completed", session.Status)
}

func assertNoop(t *testing.T, session endpoints.SessionDTO) {
	assert.Equal(t, uint64(0), session.BytesSent)
	assert.Equal(t, uint64(0), session.BytesReceived)
	assert.Equal(t, "Completed", session.Status)
}

func assertWireguard(t *testing.T, session endpoints.SessionDTO) {
	assert.Equal(t, uint64(0), session.BytesSent)
	assert.Equal(t, uint64(0), session.BytesReceived)
	assert.Equal(t, "Completed", session.Status)
}
