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
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
	hermes2ID          = "0x55fB2d361DE2aED0AbeaBfD77cA7DC8516225771"
	mystAddress        = "0x4D1d104AbD4F4351a0c51bE1e9CA0750BbCa1665"
)

var (
	providerStake, _   = big.NewInt(0).SetString("12000000000000000000", 10)
	topUpAmount, _     = big.NewInt(0).SetString("7000000000000000000", 10)
	registrationFee, _ = big.NewInt(0).SetString("100000000000000000", 10)
)

type consumer struct {
	consumerID     string
	serviceType    string
	tequila        func() *tequilapi_client.Client
	balanceSpent   *big.Int
	proposal       contract.ProposalDTO
	composeService string
	hermesID       common.Address
	hermesURL      string
}

var consumersToTest = []*consumer{
	{
		serviceType:    "noop",
		hermesID:       common.HexToAddress(hermesID),
		composeService: "myst-consumer-noop",
		hermesURL:      "http://hermes:8889/api/v2",
		tequila: func() *tequilapi_client.Client {
			return newTequilapiConsumer("myst-consumer-noop")
		},
	},
	{
		hermesID:       common.HexToAddress(hermesID),
		serviceType:    "wireguard",
		hermesURL:      "http://hermes:8889/api/v2",
		composeService: "myst-consumer-wireguard",
		tequila: func() *tequilapi_client.Client {
			return newTequilapiConsumer("myst-consumer-wireguard")
		},
	},
	{
		hermesID:       common.HexToAddress(hermesID),
		hermesURL:      "http://hermes:8889/api/v2",
		serviceType:    "openvpn",
		composeService: "myst-consumer-openvpn",
		tequila: func() *tequilapi_client.Client {
			return newTequilapiConsumer("myst-consumer-openvpn")
		},
	},
	{
		hermesURL:      "http://hermes2:8889/api/v2",
		hermesID:       common.HexToAddress(hermes2ID),
		serviceType:    "wireguard",
		composeService: "myst-consumer-hermes2",
		tequila: func() *tequilapi_client.Client {
			return newTequilapiConsumer("myst-consumer-hermes2")
		},
	},
}

func TestConsumerConnectsToProvider(t *testing.T) {
	tequilapiProvider := newTequilapiProvider()
	t.Run("Provider has a registered identity", func(t *testing.T) {
		providerRegistrationFlow(t, tequilapiProvider, providerID, providerPassphrase)
	})

	t.Run("Consumer Creates And Registers Identity", func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(len(consumersToTest))

		for _, c := range consumersToTest {
			go func(c *consumer) {
				defer wg.Done()
				tequilapiConsumer := c.tequila()
				consumerID := identityCreateFlow(t, tequilapiConsumer, consumerPassphrase)
				consumerRegistrationFlow(t, tequilapiConsumer, consumerID, consumerPassphrase)
				c.consumerID = consumerID
			}(c)
		}
		wg.Wait()
	})

	t.Run("Consumers Connect to provider", func(t *testing.T) {
		wg := sync.WaitGroup{}
		for _, c := range consumersToTest {
			wg.Add(1)
			go func(c *consumer) {
				defer wg.Done()
				proposal := consumerPicksProposal(t, c.tequila(), c.serviceType)
				balanceSpent := consumerConnectFlow(t, c.tequila(), c.consumerID, c.hermesID.Hex(), c.serviceType, proposal)
				c.balanceSpent = balanceSpent
				c.proposal = proposal
				recheckBalancesWithHermes(t, c.consumerID, balanceSpent, c.serviceType, c.hermesURL)
			}(c)
		}

		wg.Wait()
	})

	t.Run("Validate provider earnings", func(t *testing.T) {
		var sum = new(big.Int)
		for _, v := range consumersToTest {
			if v.balanceSpent == nil {
				t.Log("CONSUMER", v)
			} else {
				sum = new(big.Int).Add(sum, v.balanceSpent)
			}
		}
		providerEarnedTokens(t, tequilapiProvider, providerID, sum)
	})

	t.Run("Provider settlement flow", func(t *testing.T) {
		providerStatus, err := tequilapiProvider.Identity(providerID)
		assert.NoError(t, err)
		assert.Equal(t, new(big.Int).Sub(topUpAmount, registrationFee), providerStatus.Balance)

		// settle hermes 1
		err = tequilapiProvider.Settle(identity.FromAddress(providerID), identity.FromAddress(hermesID), true)
		assert.NoError(t, err)

		// settle hermes 2
		err = tequilapiProvider.Settle(identity.FromAddress(providerID), identity.FromAddress(hermes2ID), true)
		assert.NoError(t, err)

		providerStatus, err = tequilapiProvider.Identity(providerID)
		assert.NoError(t, err)

		fees, err := tequilapiProvider.GetTransactorFees()
		assert.NoError(t, err)

		earningsByHermes := make(map[common.Address]*big.Int)
		for _, c := range consumersToTest {
			copy := earningsByHermes[c.hermesID]

			if copy == nil {
				copy = new(big.Int)
			}

			copy = new(big.Int).Add(copy, c.balanceSpent)
			earningsByHermes[c.hermesID] = copy
		}
		hermesOneEarnings := earningsByHermes[common.HexToAddress(hermesID)]
		hermesTwoEarnings := earningsByHermes[common.HexToAddress(hermes2ID)]
		totalEarnings := new(big.Int).Add(hermesOneEarnings, hermesTwoEarnings)
		assert.Equal(t, providerStatus.EarningsTotal, totalEarnings)

		hermesFee, _ := new(big.Float).Mul(big.NewFloat(0.04), new(big.Float).SetInt(hermesOneEarnings)).Int(nil)
		feeSum := big.NewInt(0).Add(fees.Settlement, hermesFee)
		tokenSum := new(big.Int).Add(new(big.Int).Sub(topUpAmount, registrationFee), hermesOneEarnings)
		expected := new(big.Int).Sub(tokenSum, feeSum)
		diff := new(big.Int).Sub(expected, providerStatus.Balance)
		diff.Abs(diff)
		assert.True(t, diff.Uint64() >= 0 && diff.Uint64() <= 1)
	})

	t.Run("Provider decreases stake", func(t *testing.T) {
		providerStatus, err := tequilapiProvider.Identity(providerID)
		assert.NoError(t, err)
		assert.Equal(t, providerStake, providerStatus.Stake)
		initialStake := providerStatus.Stake

		fees, err := tequilapiProvider.GetTransactorFees()
		assert.NoError(t, err)

		err = tequilapiProvider.DecreaseStake(identity.FromAddress(providerID), big.NewInt(100), fees.DecreaseStake)
		assert.NoError(t, err)

		expected := new(big.Int).Sub(initialStake, big.NewInt(100))
		var lastStatus *big.Int
		assert.Eventually(t, func() bool {
			providerStatus, err := tequilapiProvider.Identity(providerID)
			assert.NoError(t, err)
			lastStatus = providerStatus.Stake
			return expected.Cmp(providerStatus.Stake) == 0
		}, time.Second*10, time.Millisecond*300, fmt.Sprintf("Incorrect stake. Expected %v, got %v", expected, lastStatus))
	})
}

func recheckBalancesWithHermes(t *testing.T, consumerID string, consumerSpending *big.Int, serviceType, hermesURL string) {
	var lastHermes *big.Int
	assert.Eventually(t, func() bool {
		hermesCaller := pingpong.NewHermesCaller(requests.NewHTTPClient("0.0.0.0", time.Second), hermesURL)
		hermesData, err := hermesCaller.GetConsumerData(consumerID)
		assert.NoError(t, err)
		promised := hermesData.LatestPromise.Amount
		lastHermes = promised
		return promised.Cmp(consumerSpending) == 0
	}, time.Second*10, time.Millisecond*300, fmt.Sprintf("Consumer reported spending %v hermes says %v. Service type %v. Hermes url %v", consumerSpending, lastHermes, serviceType, hermesURL))
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
	assert.Equal(t, new(big.Int).Sub(topUpAmount, registrationFee), idStatus.Balance)
	assert.Equal(t, providerStake, idStatus.Stake)
	assert.Zero(t, idStatus.Earnings.Uint64())
	assert.Zero(t, idStatus.EarningsTotal.Uint64())
}

func consumerRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
	err := tequilapi.Unlock(id, idPassphrase)
	assert.NoError(t, err)

	fees, err := tequilapi.GetTransactorFees()
	assert.NoError(t, err)

	err = tequilapi.RegisterIdentity(id, id, big.NewInt(0), fees.Registration)
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
	assert.Equal(t, new(big.Int).Sub(topUpAmount, registrationFee), idStatus.Balance)
	assert.Zero(t, idStatus.Earnings.Uint64())
	assert.Zero(t, idStatus.EarningsTotal.Uint64())
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

func consumerConnectFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID, hermesID, serviceType string, proposal contract.ProposalDTO) *big.Int {
	connectionStatus, err := tequilapi.ConnectionStatus()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", connectionStatus.Status)

	nonVpnIP, err := tequilapi.ConnectionIP()
	assert.NoError(t, err)
	log.Info().Msg("Original consumer IP: " + nonVpnIP.IP)

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
	log.Info().Msg("Changed consumer IP: " + vpnIP.IP)

	// sessions history should be created after connect
	sessionsDTO, err := tequilapi.SessionsByServiceType(serviceType)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sessionsDTO.Sessions))
	se := sessionsDTO.Sessions[0]
	assert.Equal(t, "e2e-land", se.ProviderCountry)
	assert.Equal(t, serviceType, se.ServiceType)
	assert.Equal(t, proposal.ProviderID, se.ProviderID)
	assert.Equal(t, connectionStatus.SessionID, se.ID)
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
	sessionsDTO, err = tequilapi.SessionsByServiceType(serviceType)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sessionsDTO.Sessions))
	se = sessionsDTO.Sessions[0]
	assert.Equal(t, "Completed", se.Status)

	// call the custom asserter for the given service type
	serviceTypeAssertionMap[serviceType](t, se)

	consumerStatus, err := tequilapi.Identity(consumerID)
	assert.NoError(t, err)
	assert.True(t, consumerStatus.Balance.Cmp(big.NewInt(0)) == 1, "consumer balance should not be empty")
	assert.True(t, consumerStatus.Balance.Cmp(new(big.Int).Sub(topUpAmount, registrationFee)) == -1, "balance should decrease but is %sd", consumerStatus.Balance)
	assert.Zero(t, consumerStatus.Earnings.Uint64())
	assert.Zero(t, consumerStatus.EarningsTotal.Uint64())

	return new(big.Int).Sub(new(big.Int).Sub(topUpAmount, registrationFee), consumerStatus.Balance)
}

func providerEarnedTokens(t *testing.T, tequilapi *tequilapi_client.Client, id string, earningsExpected *big.Int) *big.Int {
	// Before settlement
	providerStatus, err := tequilapi.Identity(id)
	assert.NoError(t, err)
	assert.True(t, providerStatus.Balance.Cmp(new(big.Int).Sub(topUpAmount, registrationFee)) == 0)
	assert.Equal(t, earningsExpected, providerStatus.Earnings, fmt.Sprintf("consumers reported spend %v, providers earnings %v", earningsExpected, providerStatus.Earnings))
	assert.Equal(t, earningsExpected, providerStatus.EarningsTotal, fmt.Sprintf("consumers reported spend %v, providers earnings %v", earningsExpected, providerStatus.Earnings))
	assert.True(t, providerStatus.Earnings.Cmp(big.NewInt(500)) == 1, "earnings should be at least 500 but is %d", providerStatus.Earnings)
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

type sessionAsserter func(t *testing.T, session contract.SessionDTO)

var serviceTypeAssertionMap = map[string]sessionAsserter{
	"openvpn": func(t *testing.T, session contract.SessionDTO) {
		assert.NotZero(t, session.Duration)
		assert.NotZero(t, session.BytesSent)
		assert.NotZero(t, session.BytesReceived)
	},
	"noop": func(t *testing.T, session contract.SessionDTO) {
		assert.NotZero(t, session.Duration)
		assert.Zero(t, session.BytesSent)
		assert.Zero(t, session.BytesReceived)
	},
	"wireguard": func(t *testing.T, session contract.SessionDTO) {
		assert.NotZero(t, session.Duration)
		assert.NotZero(t, session.BytesSent)
		assert.NotZero(t, session.BytesReceived)
	},
}
