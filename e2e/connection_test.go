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
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/tequilapi/contract"

	"github.com/mysteriumnetwork/node/session/pingpong"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
)

var (
	consumerPassphrase = "localconsumer"
	providerID         = "0xd1a23227bd5ad77f36ba62badcb78a410a1db6c5"
	providerPassphrase = "localprovider"
	hermesID           = "0xf2e2c77D2e7207d8341106E6EfA469d1940FD0d8"
)

const (
	initialBalance uint64 = 690000000
)

type consumer struct {
	consumerID   string
	serviceType  string
	tequila      *tequilapi_client.Client
	balanceSpent uint64
	proposal     contract.ProposalDTO
}

func TestConsumerConnectsToProvider(t *testing.T) {
	tequilapiProvider := newTequilapiProvider()
	t.Run("Provider has a registered identity", func(t *testing.T) {
		providerRegistrationFlow(t, tequilapiProvider, providerID, providerPassphrase)
	})

	servicesInFlag := strings.Split(*consumerServices, ",")
	consumers := make(map[string]consumer)
	t.Run("Consumer Creates And Registers Identity", func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(len(servicesInFlag))

		for _, serviceType := range servicesInFlag {
			go func(serviceType string) {
				defer wg.Done()
				tequilapiConsumer := newTequilapiConsumer(serviceType)
				consumerID := identityCreateFlow(t, tequilapiConsumer, consumerPassphrase)
				consumerRegistrationFlow(t, tequilapiConsumer, consumerID, consumerPassphrase)
				consumers[serviceType] = consumer{
					consumerID:  consumerID,
					serviceType: serviceType,
					tequila:     tequilapiConsumer,
				}
			}(serviceType)
		}

		wg.Wait()
	})

	t.Run("Consumers Connect to provider", func(t *testing.T) {
		wg := sync.WaitGroup{}
		for _, v := range consumers {
			wg.Add(1)
			go func(c consumer) {
				defer wg.Done()
				proposal := consumerPicksProposal(t, c.tequila, c.serviceType)
				balanceSpent := consumerConnectFlow(t, c.tequila, c.consumerID, hermesID, c.serviceType, proposal)
				copied := consumers[c.consumerID]
				copied.balanceSpent = balanceSpent
				copied.proposal = proposal
				consumers[c.consumerID] = copied
				recheckBalancesWithHermes(t, c.consumerID, balanceSpent, proposal.ServiceType)
			}(v)
		}
		wg.Wait()
	})

	t.Run("Validate provider earnings", func(t *testing.T) {
		var sum uint64
		for _, v := range consumers {
			sum += v.balanceSpent
		}
		providerEarnedTokens(t, tequilapiProvider, providerID, sum)
	})

	t.Run("Provider settlement flow", func(t *testing.T) {
		providerStatus, err := tequilapiProvider.Identity(providerID)
		assert.NoError(t, err)
		assert.Equal(t, initialBalance, providerStatus.Balance)

		err = tequilapiProvider.Settle(identity.FromAddress(providerID), identity.FromAddress(hermesID), true)
		assert.NoError(t, err)

		providerStatus, err = tequilapiProvider.Identity(providerID)
		assert.NoError(t, err)

		fees, err := tequilapiProvider.GetTransactorFees()
		assert.NoError(t, err)

		hermesFee := math.Round(0.04 * float64(providerStatus.EarningsTotal))
		hermesFeeUint := uint64(math.Trunc(hermesFee))

		// 10% of the settlement are used to increase provider stake.
		earnings := 0.9 * float64(providerStatus.EarningsTotal-fees.Settlement-hermesFeeUint)
		expected := initialBalance + uint64(earnings)

		// To avoid running into rounding errors, assume a delta of 2 micromyst is OK
		assert.InDelta(t, expected, providerStatus.Balance, 2)
	})
}

func recheckBalancesWithHermes(t *testing.T, consumerID string, consumerSpending uint64, serviceType string) {
	var lastHermes uint64
	assert.Eventually(t, func() bool {
		hermesCaller := pingpong.NewHermesCaller(requests.NewHTTPClient("0.0.0.0", time.Second), "http://hermes:8889/api/v2")
		hermesData, err := hermesCaller.GetConsumerData(consumerID)
		assert.NoError(t, err)
		promised := hermesData.LatestPromise.Amount
		lastHermes = promised
		return promised == consumerSpending
	}, time.Second*10, time.Millisecond*300, fmt.Sprintf("Consumer reported spending %v hermes says %v. Service type %v", consumerSpending, lastHermes, serviceType))
}

func identityCreateFlow(t *testing.T, tequilapi *tequilapi_client.Client, idPassphrase string) string {
	id, err := tequilapi.NewIdentity(idPassphrase)
	assert.NoError(t, err)
	log.Info().Msg("Created new identity: " + id.Address)

	return id.Address
}

func providerRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
	err := tequilapi.Unlock(id, idPassphrase)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		idStatus, _ := tequilapi.Identity(id)
		return "Registered" == idStatus.RegistrationStatus
	}, time.Second*30, time.Millisecond*500)

	// once we're registered, check some other information
	idStatus, err := tequilapi.Identity(id)
	assert.NoError(t, err)
	assert.Equal(t, "Registered", idStatus.RegistrationStatus)
	assert.Equal(t, "0xD4bf8ac88E7Ad1f777a084EEfD7Be4245E0b4eD3", idStatus.ChannelAddress)
	assert.Equal(t, initialBalance, idStatus.Balance)
	assert.Zero(t, idStatus.Earnings)
	assert.Zero(t, idStatus.EarningsTotal)
}

func consumerRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
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

	idStatus, err := tequilapi.Identity(id)
	assert.NoError(t, err)
	assert.Equal(t, "Registered", idStatus.RegistrationStatus)
	assert.Equal(t, initialBalance, idStatus.Balance)
	assert.Zero(t, idStatus.Earnings)
	assert.Zero(t, idStatus.EarningsTotal)
}

// expect exactly one proposal
func consumerPicksProposal(t *testing.T, tequilapi *tequilapi_client.Client, serviceType string) contract.ProposalDTO {
	var proposals []contract.ProposalDTO
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

func consumerConnectFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID, hermesID, serviceType string, proposal contract.ProposalDTO) uint64 {
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

	connectionStatus, err = tequilapi.ConnectionCreate(consumerID, proposal.ProviderID, hermesID, serviceType, contract.ConnectOptions{
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
	assert.Equal(t, "e2e-land", se.ProviderCountry)
	assert.Equal(t, serviceType, se.ServiceType)
	assert.Equal(t, proposal.ProviderID, se.ProviderID)
	assert.Equal(t, connectionStatus.SessionID, se.SessionID)
	assert.Equal(t, "New", se.Status)

	// Wait some time for session to collect stats.
	assert.Eventually(t, sessionStatsReceived(tequilapi, serviceType), 40*time.Second, 1*time.Second, serviceType)

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

	consumerStatus, err := tequilapi.Identity(consumerID)
	assert.NoError(t, err)
	assert.True(t, consumerStatus.Balance > uint64(0), "consumer balance should not be empty")
	assert.True(t, consumerStatus.Balance < uint64(690000000), "balance should decrease but is %sd", consumerStatus.Balance)
	assert.Zero(t, consumerStatus.Earnings)
	assert.Zero(t, consumerStatus.EarningsTotal)

	return uint64(690000000) - consumerStatus.Balance
}

func providerEarnedTokens(t *testing.T, tequilapi *tequilapi_client.Client, id string, earningsExpected uint64) uint64 {
	// Before settlement
	providerStatus, err := tequilapi.Identity(id)
	assert.NoError(t, err)
	assert.Equal(t, uint64(690000000), providerStatus.Balance)
	assert.Equal(t, earningsExpected, providerStatus.Earnings, fmt.Sprintf("consumers reported spend %v, providers earnings %v", earningsExpected, providerStatus.Earnings))
	assert.Equal(t, earningsExpected, providerStatus.EarningsTotal, fmt.Sprintf("consumers reported spend %v, providers earnings %v", earningsExpected, providerStatus.Earnings))
	assert.True(t, providerStatus.Earnings > uint64(500), "earnings should be at least 500 but is %d", providerStatus.Earnings)
	return providerStatus.Earnings
}

func sessionStatsReceived(tequilapi *tequilapi_client.Client, serviceType string) func() bool {
	var delegate func(stats contract.ConnectionStatisticsDTO) bool
	if serviceType != "noop" {
		delegate = func(stats contract.ConnectionStatisticsDTO) bool {
			return stats.BytesReceived > 0 && stats.BytesSent > 0 && stats.Duration > 30
		}
	} else {
		delegate = func(stats contract.ConnectionStatisticsDTO) bool {
			return stats.Duration > 30
		}
	}

	return func() bool {
		stats, err := tequilapi.ConnectionStatistics()
		if err != nil {
			return false
		}
		return delegate(stats)
	}
}

type sessionAsserter func(t *testing.T, session tequilapi_client.ConnectionSessionDTO)

var serviceTypeAssertionMap = map[string]sessionAsserter{
	"openvpn": func(t *testing.T, session tequilapi_client.ConnectionSessionDTO) {
		assert.NotZero(t, session.Duration)
		assert.NotZero(t, session.BytesSent)
		assert.NotZero(t, session.BytesReceived)
	},
	"noop": func(t *testing.T, session tequilapi_client.ConnectionSessionDTO) {
		assert.NotZero(t, session.Duration)
		assert.Zero(t, session.BytesSent)
		assert.Zero(t, session.BytesReceived)
	},
	"wireguard": func(t *testing.T, session tequilapi_client.ConnectionSessionDTO) {
		assert.NotZero(t, session.Duration)
		assert.NotZero(t, session.BytesSent)
		assert.NotZero(t, session.BytesReceived)
	},
}
