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
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
)

var (
	consumerPassphrase = "localconsumer"
	providerID         = "0xd1a23227bd5ad77f36ba62badcb78a410a1db6c5"
	providerPassphrase = "localprovider"
	accountantID       = "0xf2e2c77D2e7207d8341106E6EfA469d1940FD0d8"
)

func TestConsumerConnectsToProvider(t *testing.T) {
	tequilapiConsumer := newTequilapiConsumer()

	var consumerID string
	// no need to register provider, as he will auto-register
	t.Run("ConsumerCreatesAndRegistersIdentityFlow", func(t *testing.T) {
		consumerID = identityCreateFlow(t, tequilapiConsumer, consumerPassphrase)
		identityRegistrationFlow(t, tequilapiConsumer, consumerID, consumerPassphrase)
	})

	t.Run("ConsumerConnectFlow", func(t *testing.T) {
		servicesInFlag := strings.Split(*consumerServices, ",")
		for i, serviceType := range servicesInFlag {
			if _, ok := serviceTypeAssertionMap[serviceType]; ok {
				t.Run(serviceType, func(t *testing.T) {
					proposal := consumerPicksProposal(t, tequilapiConsumer, serviceType)
					consumerConnectFlow(t, tequilapiConsumer, consumerID, accountantID, serviceType, proposal)
				})
			}

			if i != len(servicesInFlag)-1 {
				// this allows the nats to reconnect properly before starting a new service test
				time.Sleep(time.Second * 10)
			}
		}
	})
}

func identityCreateFlow(t *testing.T, tequilapi *tequilapi_client.Client, idPassphrase string) string {
	id, err := tequilapi.NewIdentity(idPassphrase)
	assert.NoError(t, err)
	log.Info().Msg("Created new identity: " + id.Address)

	return id.Address
}

func identityRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
	err := tequilapi.Unlock(id, idPassphrase)
	assert.NoError(t, err)

	fees, err := tequilapi.GetTransactorFees()
	assert.NoError(t, err)

	err = tequilapi.RegisterIdentity(id, id, 0, fees.Registration)
	assert.NoError(t, err)

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
	err := waitForConditionFor(
		30*time.Second,
		func() (state bool, stateErr error) {
			proposals, stateErr = tequilapi.ProposalsByType(serviceType)
			return len(proposals) == 1, stateErr
		},
	)
	if err != nil {
		assert.FailNowf(t, "Exactly one proposal is expected - something is not right!", "Error was: %v", err)
	}

	log.Info().Msgf("Selected proposal is: %v, serviceType=%v", proposals[0], serviceType)
	return proposals[0]
}

func consumerConnectFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID, accountantID, serviceType string, proposal tequilapi_client.ProposalDTO) {
	err := topUpAccount(consumerID)
	assert.Nil(t, err)

	connectionStatus, err := tequilapi.ConnectionStatus()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", connectionStatus.Status)

	nonVpnIP, err := tequilapi.ConnectionIP()
	assert.NoError(t, err)
	log.Info().Msg("Original consumer IP: " + nonVpnIP)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	connectionStatus, err = tequilapi.ConnectionCreate(consumerID, proposal.ProviderID, accountantID, serviceType, tequilapi_client.ConnectOptions{
		DisableKillSwitch: false,
	})

	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus()
		return status.Status == "Connected", err
	})
	assert.NoError(t, err)

	vpnIP, err := tequilapi.ConnectionIP()
	assert.NoError(t, err)
	log.Info().Msg("Changed consumer IP: " + vpnIP)

	// sessions history should be created after connect
	sessionsDTO, err := tequilapi.ConnectionSessionsByType(serviceType)
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

	// Wait some time for session to collect stats.
	time.Sleep(2 * time.Second)

	err = tequilapi.ConnectionDestroy()
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus()
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	// sessions history should be updated after disconnect
	sessionsDTO, err = tequilapi.ConnectionSessionsByType(serviceType)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sessionsDTO.Sessions))
	se = sessionsDTO.Sessions[0]

	assert.Equal(t, "Completed", se.Status)

	// call the custom asserter for the given service type
	serviceTypeAssertionMap[serviceType](t, se)
}

type sessionAsserter func(t *testing.T, session tequilapi_client.ConnectionSessionDTO)

var serviceTypeAssertionMap = map[string]sessionAsserter{
	"openvpn": func(t *testing.T, session tequilapi_client.ConnectionSessionDTO) {
		assert.NotEqual(t, uint64(0), session.Duration)
		assert.NotEqual(t, uint64(0), session.BytesSent)
		assert.NotEqual(t, uint64(0), session.BytesReceived)
	},
	"noop": func(t *testing.T, session tequilapi_client.ConnectionSessionDTO) {
		assert.NotEqual(t, uint64(0), session.Duration)
		assert.Equal(t, uint64(0), session.BytesSent)
		assert.Equal(t, uint64(0), session.BytesReceived)
	},
	"wireguard": func(t *testing.T, session tequilapi_client.ConnectionSessionDTO) {
		assert.NotEqual(t, uint64(0), session.Duration)
		assert.NotEqual(t, uint64(0), session.BytesSent)
		assert.NotEqual(t, uint64(0), session.BytesReceived)
	},
}
