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
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/session/pingpong"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/crypto"
)

var (
	consumerPassphrase          = "localconsumer"
	providerID                  = "0xd1a23227bd5ad77f36ba62badcb78a410a1db6c5"
	providerPassphrase          = "localprovider"
	chainID               int64 = 80001
	hermesID                    = "0xd68defb97d0765741f8ecf179df2f9564e1466a3"
	hermes2ID                   = "0xfd63dc49c7163d82d6f0a4c23ff13216d702ce50"
	mystAddress                 = "0xaa9c4e723609cb913430143fbc86d3cbe7adca21"
	registryAddress             = "0x427c2bad22335710aec5e477f3e3adcd313a9bcb"
	channelImplementation       = "0x599d43715df3070f83355d9d90ae62c159e62a75"
	addressForTopups            = "0xa29fb77b25181df094908b027821a7492ca4245b"
	tenthThou                   = float64(1) / float64(10000)
)

var (
	ethClient        *ethclient.Client
	ethClientL2      *ethclient.Client
	ethSignerBuilder func(client *ethclient.Client) func(address common.Address, tx *types.Transaction) (*types.Transaction, error)
)

var (
	providerChannelAddress      = "0xa2305c7214045100B6EF5Df8f8FEc5C57F42051A"
	balanceAfterRegistration, _ = big.NewInt(0).SetString("7000000000000000000", 10)
	registrationFee, _          = big.NewInt(0).SetString("100000000000000000", 10)
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
	initEthClient(t)

	tequilapiProvider := newTequilapiProvider()
	t.Run("Provider has a registered identity", func(t *testing.T) {
		providerRegistrationFlow(t, tequilapiProvider, providerID, providerPassphrase)
	})

	t.Run("Consumer Creates And Registers Identity", func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(len(consumersToTest))
		topUps := make(chan *consumer, len(consumersToTest))
		defer close(topUps)

		go func() {
			for c := range topUps {
				fees, err := c.tequila().GetTransactorFees()
				assert.NoError(t, err)
				topUpConsumer(t, c.consumerID, c.hermesID, fees.Registration)
			}
		}()

		for _, c := range consumersToTest {
			go func(c *consumer) {
				defer wg.Done()
				tequilapiConsumer := c.tequila()
				consumerID := identityCreateFlow(t, tequilapiConsumer, consumerPassphrase)
				c.consumerID = consumerID
				topUps <- c
				consumerRegistrationFlow(t, tequilapiConsumer, consumerID, consumerPassphrase)
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
		sum := new(big.Int)
		for _, v := range consumersToTest {
			sum = new(big.Int).Add(sum, v.balanceSpent)
		}
		providerEarnedTokens(t, tequilapiProvider, providerID, sum)
	})

	t.Run("Provider settlement flow", func(t *testing.T) {
		caller, err := bindings.NewMystTokenCaller(common.HexToAddress(mystAddress), ethClient)
		assert.NoError(t, err)

		providerStatus, err := tequilapiProvider.Identity(providerID)
		assert.NoError(t, err)
		assert.Equal(t, balanceAfterRegistration, providerStatus.Balance)

		// try to settle hermes 1
		hermeses := []common.Address{
			common.HexToAddress(hermesID),
		}
		err = tequilapiProvider.Settle(identity.FromAddress(providerID), hermeses, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is set as beneficiary, skip settling")

		hermeses = append(hermeses, common.HexToAddress(hermes2ID))
		beneficiary := "0x1234aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa123412"
		err = tequilapiProvider.SetBeneficiaryAsync(providerID, beneficiary)
		assert.NoError(t, err)

		// settle hermes 1 and 2 (should settle with beneficiary)
		err = tequilapiProvider.Settle(identity.FromAddress(providerID), hermeses, true)
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
		fdiff := getDiffFloat(providerStatus.EarningsTotal, totalEarnings)
		assert.True(t, fdiff < tenthThou)

		hic, err := bindings.NewHermesImplementationCaller(common.HexToAddress(hermesID), ethClient)
		assert.NoError(t, err)

		hermesFee, err := hic.CalculateHermesFee(&bind.CallOpts{}, totalEarnings)
		assert.NoError(t, err)

		feeSum := big.NewInt(0).Add(big.NewInt(0).Add(fees.Settlement, hermesFee), fees.Settlement)
		expected := new(big.Int).Sub(totalEarnings, feeSum)
		balance, err := caller.BalanceOf(&bind.CallOpts{}, common.HexToAddress(beneficiary))
		assert.NoError(t, err)

		diff := getDiffFloat(balance, expected)
		diff = math.Abs(diff)

		assert.True(t, diff >= 0 && diff <= tenthThou, fmt.Sprintf("got diff %v", diff))
	})

	t.Run("Provider withdraws to l1", func(t *testing.T) {
		// since we've changed the benef, our channel is empty. Pretend that we do have myst in it.
		chid, err := crypto.GenerateChannelAddress(providerID, hermes2ID, registryAddress, channelImplementation)
		assert.NoError(t, err)
		mintMyst(t, crypto.FloatToBigMyst(1), common.HexToAddress(chid), ethClientL2)

		beneficiary := common.HexToAddress("0x1231adadadadaadada123123")
		caller, err := bindings.NewMystTokenCaller(common.HexToAddress(mystAddress), ethClient)
		assert.NoError(t, err)

		balance, err := caller.BalanceOf(&bind.CallOpts{}, beneficiary)
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(0).Uint64(), balance.Uint64())

		err = tequilapiProvider.Withdraw(identity.FromAddress(providerID), common.HexToAddress(hermes2ID), beneficiary, nil, chainID, 0)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			balance, _ := caller.BalanceOf(&bind.CallOpts{}, beneficiary)
			return balance.Cmp(big.NewInt(0)) == 1
		}, time.Second*20, time.Millisecond*100)
	})

	t.Run("Provider stops services", func(t *testing.T) {
		services, err := tequilapiProvider.Services()
		assert.NoError(t, err)

		for _, service := range services {
			err := tequilapiProvider.ServiceStop(service.ID)
			assert.NoError(t, err)
		}
	})

	t.Run("Provider starts whitelisted noop services", func(t *testing.T) {
		req := contract.ServiceStartRequest{
			ProviderID:     providerID,
			Type:           "noop",
			AccessPolicies: &contract.ServiceAccessPolicies{IDs: []string{"mysterium"}},
		}

		_, err := tequilapiProvider.ServiceStart(req)
		assert.NoError(t, err)
	})

	t.Run("Whitelisted consumer connects to the whitelisted noop service", func(t *testing.T) {
		c := consumersToTest[0]

		topUpConsumer(t, "0xc4cb9a91b8498776f6f8a0d5a2a23beec9b3cef3", common.HexToAddress(hermesID), registrationFee)

		consumerRegistrationFlow(t, c.tequila(), "0xc4cb9a91b8498776f6f8a0d5a2a23beec9b3cef3", "")

		proposal := consumerPicksProposal(t, c.tequila(), "noop")
		consumerConnectFlow(t, c.tequila(), "0xc4cb9a91b8498776f6f8a0d5a2a23beec9b3cef3", hermesID, "noop", proposal)
	})

	t.Run("Consumers rejected by whitelisted service", func(t *testing.T) {
		c := consumersToTest[0]
		proposal := consumerPicksProposal(t, c.tequila(), c.serviceType)
		consumerRejectWhitelistedFlow(t, c.tequila(), c.consumerID, hermesID, c.serviceType, proposal)
	})

	t.Run("Registration with bounty", func(t *testing.T) {
		t.Run("consumer", func(t *testing.T) {
			c := consumersToTest[0]
			id, err := c.tequila().NewIdentity("")
			assert.NoError(t, err)

			err = c.tequila().Unlock(id.Address, "")
			assert.NoError(t, err)

			status, err := c.tequila().IdentityRegistrationStatus(id.Address)
			assert.NoError(t, err)
			assert.Equal(t, "Unregistered", status.Status)

			err = c.tequila().RegisterIdentity(id.Address, "", nil)
			assert.NoError(t, err)

			assert.Eventually(t, func() bool {
				status, err := c.tequila().IdentityRegistrationStatus(id.Address)
				if err != nil {
					return false
				}
				return status.Status == "Registered"
			}, time.Second*20, time.Millisecond*100)
		})
		t.Run("provider", func(t *testing.T) {
			c := consumersToTest[0]
			id, err := c.tequila().NewIdentity("")
			assert.NoError(t, err)

			err = c.tequila().Unlock(id.Address, "")
			assert.NoError(t, err)

			status, err := c.tequila().IdentityRegistrationStatus(id.Address)
			assert.NoError(t, err)
			assert.Equal(t, "Unregistered", status.Status)

			err = c.tequila().RegisterIdentity(id.Address, "", nil)
			assert.NoError(t, err)

			assert.Eventually(t, func() bool {
				status, err := c.tequila().IdentityRegistrationStatus(id.Address)
				if err != nil {
					return false
				}
				return status.Status == "Registered"
			}, time.Second*20, time.Millisecond*100)
		})
	})
}

func recheckBalancesWithHermes(t *testing.T, consumerID string, consumerSpending *big.Int, serviceType, hermesURL string) {
	var testSuccess bool
	var lastHermes *big.Int
	assert.Eventually(t, func() bool {
		hermesCaller := pingpong.NewHermesCaller(requests.NewHTTPClient("0.0.0.0", time.Second), hermesURL)
		hermesData, err := hermesCaller.GetConsumerData(chainID, consumerID, -1)
		assert.NoError(t, err)
		promised := hermesData.LatestPromise.Amount
		lastHermes = promised

		// Author: vkuznecovas
		// Due to the async nature of the payment system, a situation might occur where a consumers reported spending is larger than the actual amount that reaches hermes.
		// This happens in the following flow:
		// 1) Session is established.
		// 2) Payments occur normally for some time.
		// 3) Consumer received yet another invoice.
		// 4) Consumer starts calculating the amount to promise.
		// 5) Session is killed as operation 4) is happening.
		// 6) The promise is incremented but never issued as the session is aborted. This can happen due to promise not being sent, or provider not listening for further promises on the given topic.
		// 7) Hermes is not aware of the last promise, therefore the consumer and hermes reportings are different.
		// I do not believe this will cause issues in reality, as such situations occur in e2e test regularly due to the rapid exchange of promises.
		// Under normal circumstances, such occurences should be VERY, VERY rare and the amount of myst involved is rather small. They should have no impact on the payment system as a whole.
		// Therefore, for these tests to be stable, the following solution is proposed:
		// Make sure that the hermes and consumer reported spendings differ by no more than 1/100000 of a myst.
		absDiffFloat := getDiffFloat(consumerSpending, promised)
		res := absDiffFloat < tenthThou
		if res {
			testSuccess = true
		}
		return res
	}, time.Second*20, time.Millisecond*300)

	if !testSuccess {
		fmt.Printf("Consumer reported spending %v hermes says %v. Service type %v. Hermes url %v, consumer ID %v", consumerSpending, lastHermes, serviceType, hermesURL, consumerID)
	}
}

func getDiffFloat(a, b *big.Int) float64 {
	diff := new(big.Int).Sub(a, b)
	absoluteDiff := new(big.Int).Abs(diff)
	return crypto.BigMystToFloat(absoluteDiff)
}

func identityCreateFlow(t *testing.T, tequilapi *tequilapi_client.Client, idPassphrase string) string {
	id, err := tequilapi.NewIdentity(idPassphrase)
	assert.NoError(t, err)
	log.Info().Msg("Created new identity: " + id.Address)

	return id.Address
}

func initEthClient(t *testing.T) {
	addr := common.HexToAddress(addressForTopups)
	ks := keystore.NewKeyStore("/node/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.Find(accounts.Account{Address: addr})
	assert.NoError(t, err)

	err = ks.Unlock(acc, "")
	assert.NoError(t, err)

	c, err := ethclient.Dial("http://ganache:8545")
	assert.NoError(t, err)

	ethClient = c

	c2, err := ethclient.Dial("ws://ganache2:8545")
	assert.NoError(t, err)
	ethClientL2 = c2

	ethSignerBuilder = func(client *ethclient.Client) func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		chainId, err := client.ChainID(context.Background())
		if err != nil {
			return func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
				return nil, err
			}
		}
		return func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return ks.SignTx(acc, tx, chainId)
		}
	}

}

func mintMyst(t *testing.T, amount *big.Int, chid common.Address, ethc *ethclient.Client) {
	ts, err := bindings.NewMystTokenTransactor(common.HexToAddress(mystAddress), ethc)
	assert.NoError(t, err)

	nonce, err := ethc.PendingNonceAt(context.Background(), common.HexToAddress(addressForTopups))
	assert.NoError(t, err)

	_, err = ts.Transfer(&bind.TransactOpts{
		From:   common.HexToAddress(addressForTopups),
		Signer: ethSignerBuilder(ethc),
		Nonce:  big.NewInt(0).SetUint64(nonce),
	}, chid, amount)
	assert.NoError(t, err)
}

func providerRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
	err := tequilapi.Unlock(id, idPassphrase)
	assert.NoError(t, err)

	err = tequilapi.RegisterIdentity(id, "", nil)
	assert.True(t, err == nil || assert.Contains(t, err.Error(), "registration in progress"))

	assert.Eventually(t, func() bool {
		idStatus, _ := tequilapi.Identity(id)
		return idStatus.RegistrationStatus == "Registered"
	}, time.Second*30, time.Millisecond*500)

	// once we're registered, check some other information
	idStatus, err := tequilapi.Identity(id)
	assert.NoError(t, err)

	assert.Equal(t, "Registered", idStatus.RegistrationStatus)
	assert.Equal(t, providerChannelAddress, idStatus.ChannelAddress)
	assert.Eventually(t, func() bool {
		balance, err := tequilapi.BalanceRefresh(id)
		assert.NoError(t, err)
		return balanceAfterRegistration.Cmp(balance.Balance) == 0
	}, time.Second*20, time.Millisecond*500)
	assert.Zero(t, idStatus.Earnings.Uint64())
	assert.Zero(t, idStatus.EarningsTotal.Uint64())
}

func topUpConsumer(t *testing.T, id string, hermesID common.Address, registrationFee *big.Int) {
	// TODO: once free registration is a thing of the past, remove this return

	// chid, err := crypto.GenerateChannelAddress(id, hermesID.Hex(), registryAddress, channelImplementation)
	// assert.NoError(t, err)

	// // add some balance for fees + consuming service
	// amountToTopUp := big.NewInt(0).Mul(registrationFee, big.NewInt(20))
	// mintMyst(t, amountToTopUp, common.HexToAddress(chid))
}

func consumerRegistrationFlow(t *testing.T, tequilapi *tequilapi_client.Client, id, idPassphrase string) {
	err := tequilapi.Unlock(id, idPassphrase)
	assert.NoError(t, err)

	err = tequilapi.RegisterIdentity(id, "", nil)
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
	assert.Eventually(t, func() bool {
		balance, err := tequilapi.BalanceRefresh(id)
		assert.NoError(t, err)
		return balanceAfterRegistration.Cmp(balance.Balance) == 0
	}, time.Second*20, time.Millisecond*500)
	assert.Zero(t, idStatus.Earnings.Uint64())
	assert.Zero(t, idStatus.EarningsTotal.Uint64())
}

// expect exactly one proposal
func consumerPicksProposal(t *testing.T, tequilapi *tequilapi_client.Client, serviceType string) contract.ProposalDTO {
	var proposals []contract.ProposalDTO
	assert.Eventually(t, func() bool {
		p, stateErr := tequilapi.ProposalsByTypeWithWhitelisting(serviceType)
		if stateErr != nil {
			log.Err(stateErr)
			return false
		}
		proposals = p
		return len(p) == 1
	}, time.Second*30, time.Millisecond*200)

	log.Info().Msgf("Selected proposal is: %v, serviceType=%v", proposals[0], serviceType)
	return proposals[0]
}

func consumerConnectFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID, hermesID, serviceType string, proposal contract.ProposalDTO) *big.Int {
	connectionStatus, err := tequilapi.ConnectionStatus(0)
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", connectionStatus.Status)

	_, err = tequilapi.ConnectionIP()
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus(0)
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	connectionStatus, err = tequilapi.ConnectionCreate(consumerID, proposal.ProviderID, hermesID, serviceType, contract.ConnectOptions{
		DisableKillSwitch: false,
	})

	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus(0)
		return status.Status == "Connected", err
	})
	assert.NoError(t, err)

	_, err = tequilapi.ConnectionIP()
	assert.NoError(t, err)

	// sessions history should be created after connect
	sessionsDTO, err := tequilapi.SessionsByServiceType(serviceType)
	assert.NoError(t, err)

	require.True(t, len(sessionsDTO.Items) >= 1)
	se := sessionsDTO.Items[0]
	assert.Equal(t, "e2e-land", se.ProviderCountry)
	assert.Equal(t, serviceType, se.ServiceType)
	assert.Equal(t, proposal.ProviderID, se.ProviderID)
	assert.Equal(t, connectionStatus.SessionID, se.ID)
	assert.Equal(t, "New", se.Status)

	// Wait some time for session to collect stats.
	assert.Eventually(t, sessionStatsReceived(tequilapi, serviceType), 60*time.Second, 1*time.Second, serviceType)

	err = tequilapi.ConnectionDestroy(0)
	assert.NoError(t, err)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus(0)
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	// sessions history should be updated after disconnect
	sessionsDTO, err = tequilapi.SessionsByServiceType(serviceType)
	assert.NoError(t, err)

	assert.True(t, len(sessionsDTO.Items) >= 1)
	se = sessionsDTO.Items[0]
	assert.Equal(t, "Completed", se.Status)

	// call the custom asserter for the given service type
	serviceTypeAssertionMap[serviceType](t, se)
	consumerStatus := contract.IdentityDTO{}
	assert.Eventually(t, func() bool {
		cs, err := tequilapi.Identity(consumerID)
		if err != nil {
			return false
		}
		consumerStatus = cs
		return true
	}, time.Second*20, time.Millisecond*150)
	assert.True(t, consumerStatus.Balance.Cmp(big.NewInt(0)) == 1, "consumer balance should not be empty")
	assert.True(t, consumerStatus.Balance.Cmp(balanceAfterRegistration) == -1, "balance should decrease but is %s", consumerStatus.Balance)
	assert.Zero(t, consumerStatus.Earnings.Uint64())
	assert.Zero(t, consumerStatus.EarningsTotal.Uint64())

	return new(big.Int).Sub(balanceAfterRegistration, consumerStatus.Balance)
}

func consumerRejectWhitelistedFlow(t *testing.T, tequilapi *tequilapi_client.Client, consumerID, accountantID, serviceType string, proposal contract.ProposalDTO) {
	connectionStatus, err := tequilapi.ConnectionStatus(0)
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", connectionStatus.Status)

	nonVpnIP, err := tequilapi.ConnectionIP()
	assert.NoError(t, err)
	log.Info().Msg("Original consumer IP: " + nonVpnIP.IP)

	err = waitForCondition(func() (bool, error) {
		status, err := tequilapi.ConnectionStatus(0)
		return status.Status == "NotConnected", err
	})
	assert.NoError(t, err)

	_, err = tequilapi.ConnectionCreate(consumerID, proposal.ProviderID, accountantID, serviceType, contract.ConnectOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "consumer identity is not allowed")
}

func providerEarnedTokens(t *testing.T, tequilapi *tequilapi_client.Client, id string, earningsExpected *big.Int) *big.Int {
	// Before settlement
	assert.Eventually(t, func() bool {
		providerStatus, err := tequilapi.Identity(id)
		if err != nil {
			return false
		}

		fdiff := getDiffFloat(providerStatus.Earnings, earningsExpected)
		return fdiff < tenthThou
	}, time.Second*20, time.Millisecond*250)

	var providerStatus contract.IdentityDTO
	var err error
	assert.Eventually(t, func() bool {
		providerStatus, err = tequilapi.Identity(id)
		assert.NoError(t, err)
		return providerStatus.Balance.Cmp(balanceAfterRegistration) == 0
	}, time.Second*20, time.Millisecond*500)

	// For reasoning behind these, see the comment in recheckBalancesWithHermes
	actualEarnings := getDiffFloat(earningsExpected, providerStatus.Earnings)
	assert.True(t, actualEarnings < tenthThou)

	actualEarnings = getDiffFloat(earningsExpected, providerStatus.EarningsTotal)
	assert.True(t, actualEarnings < tenthThou)

	assert.True(t, providerStatus.Earnings.Cmp(big.NewInt(500)) == 1, "earnings should be at least 500 but is %d", providerStatus.Earnings)
	return providerStatus.Earnings
}

func sessionStatsReceived(tequilapi *tequilapi_client.Client, serviceType string) func() bool {
	var delegate func(stats contract.ConnectionStatisticsDTO) bool
	if serviceType != "noop" {
		delegate = func(stats contract.ConnectionStatisticsDTO) bool {
			return stats.BytesReceived > 0 && stats.BytesSent > 0 && stats.Duration > 45
		}
	} else {
		delegate = func(stats contract.ConnectionStatisticsDTO) bool {
			return stats.Duration > 45
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
