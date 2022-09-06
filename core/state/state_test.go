/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package state

import (
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	nodeSession "github.com/mysteriumnetwork/node/session"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/session/pingpong"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/payments/crypto"
)

type debounceTester struct {
	numInteractions int
	lock            sync.Mutex
}

type interactionCounter interface {
	interactions() int
}

func (dt *debounceTester) do(interface{}) {
	dt.lock.Lock()
	dt.numInteractions++
	dt.lock.Unlock()
}

func (dt *debounceTester) interactions() int {
	dt.lock.Lock()
	defer dt.lock.Unlock()
	return dt.numInteractions
}

func Test_Debounce_CallsOnceInInterval(t *testing.T) {
	dt := &debounceTester{}
	duration := time.Millisecond * 10
	f := debounce(dt.do, duration)
	for i := 1; i < 10; i++ {
		f(struct{}{})
	}
	assert.Eventually(t, interacted(dt, 1), 2*time.Second, 10*time.Millisecond)
}

type mockPublisher struct {
	lock           sync.Mutex
	publishedTopic string
	publishedData  interface{}
}

func (mp *mockPublisher) Publish(topic string, data interface{}) {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	mp.publishedData = data
	mp.publishedTopic = topic
}

type serviceListerMock struct {
	lock             sync.Mutex
	numInteractions  int
	servicesToReturn map[service.ID]*service.Instance
}

func (slm *serviceListerMock) interactions() int {
	slm.lock.Lock()
	defer slm.lock.Unlock()
	return slm.numInteractions
}

func (slm *serviceListerMock) List(includeAll bool) []*service.Instance {
	slm.lock.Lock()
	defer slm.lock.Unlock()
	slm.numInteractions++
	list := make([]*service.Instance, 0, len(slm.servicesToReturn))
	for _, instance := range slm.servicesToReturn {
		list = append(list, instance)
	}
	return list
}

func Test_ConsumesSessionEvents(t *testing.T) {
	// given
	expected := sessionEvent.SessionContext{
		ID:         "1",
		StartedAt:  time.Now(),
		ConsumerID: identity.FromAddress("0x0000000000000000000000000000000000000001"),
		HermesID:   common.HexToAddress("0x000000000000000000000000000000000000000a"),
		Proposal: market.ServiceProposal{
			Location: stubLocation,
		},
	}

	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	keeper.Subscribe(eventBus)

	// when
	eventBus.Publish(sessionEvent.AppTopicSession, sessionEvent.AppEventSession{
		Status:  sessionEvent.CreatedStatus,
		Session: expected,
	})

	// then
	assert.Eventually(t, func() bool {
		return len(keeper.GetState().Sessions) == 1
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(
		t,
		[]session.History{
			{
				SessionID:       nodeSession.ID(expected.ID),
				Direction:       session.DirectionProvided,
				ConsumerID:      expected.ConsumerID,
				HermesID:        expected.HermesID.Hex(),
				ProviderCountry: "MU",
				Started:         expected.StartedAt,
				Status:          session.StatusNew,
				Tokens:          new(big.Int),
			},
		},
		keeper.GetState().Sessions,
	)

	// when
	eventBus.Publish(sessionEvent.AppTopicSession, sessionEvent.AppEventSession{
		Status:  sessionEvent.RemovedStatus,
		Session: expected,
	})

	// then
	assert.Eventually(t, func() bool {
		return len(keeper.GetState().Sessions) == 0
	}, 2*time.Second, 10*time.Millisecond)
}

func Test_ConsumesSessionAcknowledgeEvents(t *testing.T) {
	// given
	myID := "test"
	expected := session.History{
		SessionID: nodeSession.ID("1"),
	}

	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	keeper.Subscribe(eventBus)
	keeper.state.Services = []contract.ServiceInfoDTO{
		{ID: myID, ConnectionStatistics: &contract.ServiceStatisticsDTO{}},
	}
	keeper.state.Sessions = []session.History{
		expected,
	}

	// when
	eventBus.Publish(sessionEvent.AppTopicSession, sessionEvent.AppEventSession{
		Status: sessionEvent.AcknowledgedStatus,
		Service: sessionEvent.ServiceContext{
			ID: myID,
		},
		Session: sessionEvent.SessionContext{
			ID: string(expected.SessionID),
		},
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetState().Services[0].ConnectionStatistics.Successful == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func Test_consumeServiceSessionEarningsEvent(t *testing.T) {
	// given
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	keeper.Subscribe(eventBus)
	keeper.state.Sessions = []session.History{
		{SessionID: nodeSession.ID("1")},
	}

	// when
	eventBus.Publish(sessionEvent.AppTopicTokensEarned, sessionEvent.AppEventTokensEarned{
		SessionID: "1",
		Total:     big.NewInt(500),
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetState().Sessions[0].Tokens.Cmp(big.NewInt(0)) != 0
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(
		t,
		[]session.History{
			{SessionID: nodeSession.ID("1"), Tokens: big.NewInt(500)},
		},
		keeper.GetState().Sessions,
	)
}

func Test_consumeServiceSessionStatisticsEvent(t *testing.T) {
	// given
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	keeper.Subscribe(eventBus)
	keeper.state.Sessions = []session.History{
		{SessionID: nodeSession.ID("1")},
	}

	// when
	eventBus.Publish(sessionEvent.AppTopicDataTransferred, sessionEvent.AppEventDataTransferred{
		ID:   "1",
		Up:   1,
		Down: 2,
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetState().Sessions[0].DataReceived != 0
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(
		t,
		[]session.History{
			{SessionID: "1", DataSent: 2, DataReceived: 1},
		},
		keeper.GetState().Sessions,
	)
}

func Test_ConsumesServiceEvents(t *testing.T) {
	mpr := mockProposalRepository{
		priceToAdd: market.Price{
			PricePerHour: big.NewInt(1),
			PricePerGiB:  big.NewInt(2),
		},
	}

	expected := service.Instance{
		Proposal: market.NewProposal("0xbeef", "wireguard", market.NewProposalOpts{}),
	}
	var id service.ID

	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{
			id: &expected,
		},
	}

	duration := time.Millisecond * 3
	deps := KeeperDeps{
		Publisher:        publisher,
		ServiceLister:    sl,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
		ProposalPricer:   &mpr,
	}
	keeper := NewKeeper(deps, duration)

	for i := 0; i < 5; i++ {
		// shoot a few events to see if we'll debounce
		keeper.consumeServiceStateEvent(servicestate.AppEventServiceStatus{})
	}

	assert.Eventually(t, interacted(sl, 1), 2*time.Second, 10*time.Millisecond)

	actual := keeper.GetState().Services[0]
	assert.Equal(t, string(id), actual.ID)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.ProviderID.Address, actual.ProviderID)
	assert.Equal(t, expected.Options, actual.Options)
	assert.Equal(t, string(expected.State()), actual.Status)
	expt, _ := mpr.EnrichProposalWithPrice(expected.Proposal)
	assert.EqualValues(t, contract.NewProposalDTO(expt), *actual.Proposal)
}

func Test_ConsumesConnectionStateEvents(t *testing.T) {
	// given
	expected := connectionstate.Status{
		State:     connectionstate.Connected,
		SessionID: "1",
		Proposal: proposal.PricedServiceProposal{
			ServiceProposal: market.ServiceProposal{
				Contacts: market.ContactList{},
			},
		},
	}
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		ServiceLister:    &serviceListerMock{},
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	err := keeper.Subscribe(eventBus)
	assert.NoError(t, err)
	assert.Equal(t, connectionstate.NotConnected, keeper.GetConnection("1").Session.State)

	// when
	eventBus.Publish(connectionstate.AppTopicConnectionState, connectionstate.AppEventConnectionState{
		State:       expected.State,
		SessionInfo: expected,
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetConnection("1").Session.State == connectionstate.Connected
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, expected, keeper.GetConnection("1").Session)
}

func Test_ConsumesConnectionStatisticsEvents(t *testing.T) {
	// given
	expected := connectionstate.Statistics{
		At:            time.Now(),
		BytesReceived: 10 * datasize.MiB.Bytes(),
		BytesSent:     500 * datasize.KiB.Bytes(),
	}
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		ServiceLister:    &serviceListerMock{},
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	err := keeper.Subscribe(eventBus)
	assert.NoError(t, err)
	assert.True(t, keeper.GetConnection("").Statistics.At.IsZero())

	// when
	eventBus.Publish(connectionstate.AppTopicConnectionStatistics, connectionstate.AppEventConnectionStatistics{
		Stats: expected,
	})

	// then
	assert.Eventually(t, func() bool {
		return expected == keeper.GetConnection("").Statistics
	}, 2*time.Second, 10*time.Millisecond)
}

func Test_ConsumesConnectionInvoiceEvents(t *testing.T) {
	// given
	expected := crypto.Invoice{
		AgreementID:    big.NewInt(1),
		AgreementTotal: big.NewInt(1001),
		TransactorFee:  big.NewInt(10),
	}
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:        eventBus,
		ServiceLister:    &serviceListerMock{},
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	err := keeper.Subscribe(eventBus)
	assert.NoError(t, err)
	assert.True(t, keeper.GetConnection("").Statistics.At.IsZero())

	// when
	eventBus.Publish(pingpongEvent.AppTopicInvoicePaid, pingpongEvent.AppEventInvoicePaid{
		Invoice: expected,
	})

	// then
	assert.Eventually(t, func() bool {
		return reflect.DeepEqual(expected, keeper.GetConnection("").Invoice)
	}, 2*time.Second, 10*time.Millisecond)
}

func Test_ConsumesBalanceChangeEvent(t *testing.T) {
	// given
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:     eventBus,
		ServiceLister: &serviceListerMock{},
		IdentityProvider: &mocks.IdentityProvider{
			Identities: []identity.Identity{
				{Address: "0x000000000000000000000000000000000000000a"},
			},
		},
		IdentityRegistry:          &mocks.IdentityRegistry{Status: registry.Registered},
		IdentityChannelCalculator: &mockChannelAddressCalculator{},
		BalanceProvider:           &mockBalanceProvider{Balance: big.NewInt(0)},
		EarningsProvider:          &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	err := keeper.Subscribe(eventBus)
	assert.NoError(t, err)
	assert.Zero(t, keeper.GetState().Identities[0].Balance.Uint64())

	// when
	eventBus.Publish(pingpongEvent.AppTopicBalanceChanged, pingpongEvent.AppEventBalanceChanged{
		Identity: identity.Identity{Address: "0x000000000000000000000000000000000000000a"},
		Previous: big.NewInt(0),
		Current:  big.NewInt(999),
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetState().Identities[0].Balance.Cmp(big.NewInt(999)) == 0
	}, 2*time.Second, 10*time.Millisecond)
}

func Test_ConsumesEarningsChangeEvent(t *testing.T) {
	// given
	eventBus := eventbus.New()
	channelsProvider := &mockEarningsProvider{
		Channels: []pingpong.HermesChannel{
			{ChannelID: "1"},
		},
	}

	deps := KeeperDeps{
		Publisher:     eventBus,
		ServiceLister: &serviceListerMock{},
		IdentityProvider: &mocks.IdentityProvider{
			Identities: []identity.Identity{
				{Address: "0x000000000000000000000000000000000000000a"},
			},
		},
		IdentityRegistry:          &mocks.IdentityRegistry{Status: registry.Registered},
		IdentityChannelCalculator: &mockChannelAddressCalculator{},
		BalanceProvider:           &mockBalanceProvider{Balance: big.NewInt(0)},
		EarningsProvider:          channelsProvider,
	}
	keeper := NewKeeper(deps, time.Millisecond)
	err := keeper.Subscribe(eventBus)
	assert.NoError(t, err)
	assert.Zero(t, keeper.GetState().Identities[0].Balance.Uint64())
	assert.Len(t, keeper.GetState().ProviderChannels, 1)
	assert.Equal(t, channelsProvider.Channels, keeper.GetState().ProviderChannels)

	// when
	channelsProvider.Channels = []pingpong.HermesChannel{
		{ChannelID: "1"},
		{ChannelID: "2"},
	}
	eventBus.Publish(pingpongEvent.AppTopicEarningsChanged, pingpongEvent.AppEventEarningsChanged{
		Identity: identity.Identity{Address: "0x000000000000000000000000000000000000000a"},
		Previous: pingpongEvent.EarningsDetailed{},
		Current: pingpongEvent.EarningsDetailed{
			Total: pingpongEvent.Earnings{
				LifetimeBalance:  big.NewInt(100),
				UnsettledBalance: big.NewInt(10),
			},
		},
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetState().Identities[0].Earnings.Cmp(big.NewInt(10)) == 0 && keeper.GetState().Identities[0].EarningsTotal.Cmp(big.NewInt(100)) == 0
	}, 2*time.Second, 10*time.Millisecond)
	assert.Len(t, keeper.GetState().ProviderChannels, 2)
	assert.Equal(t, channelsProvider.Channels, keeper.GetState().ProviderChannels)
}

func Test_ConsumesIdentityRegistrationEvent(t *testing.T) {
	// given
	eventBus := eventbus.New()
	deps := KeeperDeps{
		Publisher:     eventBus,
		ServiceLister: &serviceListerMock{},
		IdentityProvider: &mocks.IdentityProvider{
			Identities: []identity.Identity{
				{Address: "0x000000000000000000000000000000000000000a"},
			},
		},
		IdentityRegistry:          &mocks.IdentityRegistry{Status: registry.Unregistered},
		IdentityChannelCalculator: &mockChannelAddressCalculator{},
		BalanceProvider:           &mockBalanceProvider{Balance: big.NewInt(0)},
		EarningsProvider:          &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, time.Millisecond)
	err := keeper.Subscribe(eventBus)
	assert.NoError(t, err)
	assert.Equal(t, registry.Unregistered, keeper.GetState().Identities[0].RegistrationStatus)

	// when
	eventBus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:     identity.Identity{Address: "0x000000000000000000000000000000000000000a"},
		Status: registry.Registered,
	})

	// then
	assert.Eventually(t, func() bool {
		return keeper.GetState().Identities[0].RegistrationStatus == registry.Registered
	}, 2*time.Second, 10*time.Millisecond)
}

func Test_getServiceByID(t *testing.T) {
	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{},
	}

	duration := time.Millisecond * 3
	deps := KeeperDeps{
		Publisher:        publisher,
		ServiceLister:    sl,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, duration)
	myID := "test"
	keeper.state.Services = []contract.ServiceInfoDTO{
		{ID: myID},
		{ID: "mock"},
	}

	s, found := keeper.getServiceByID(myID)
	assert.True(t, found)

	assert.EqualValues(t, keeper.state.Services[0], s)

	_, found = keeper.getServiceByID("something else")
	assert.False(t, found)
}

func Test_incrementConnectionCount(t *testing.T) {
	expected := service.Instance{}
	var id service.ID

	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{
			id: &expected,
		},
	}

	duration := time.Millisecond * 3
	deps := KeeperDeps{
		Publisher:        publisher,
		ServiceLister:    sl,
		IdentityProvider: &mocks.IdentityProvider{},
		EarningsProvider: &mockEarningsProvider{},
	}
	keeper := NewKeeper(deps, duration)
	myID := "test"
	keeper.state.Services = []contract.ServiceInfoDTO{
		{ID: myID, ConnectionStatistics: &contract.ServiceStatisticsDTO{}},
		{ID: "mock", ConnectionStatistics: &contract.ServiceStatisticsDTO{}},
	}

	keeper.incrementConnectCount(myID, false)
	s, found := serviceByID(keeper.GetState().Services, myID)
	assert.True(t, found)

	assert.Equal(t, 1, s.ConnectionStatistics.Attempted)
	assert.Equal(t, 0, s.ConnectionStatistics.Successful)

	keeper.incrementConnectCount(myID, true)
	s, found = serviceByID(keeper.GetState().Services, myID)
	assert.True(t, found)

	assert.Equal(t, 1, s.ConnectionStatistics.Successful)
	assert.Equal(t, 1, s.ConnectionStatistics.Attempted)
}

func interacted(c interactionCounter, times int) func() bool {
	return func() bool {
		return c.interactions() == times
	}
}

type mockBalanceProvider struct {
	Balance *big.Int
}

// GetBalance returns a pre-defined balance.
func (mbp *mockBalanceProvider) GetBalance(_ int64, _ identity.Identity) *big.Int {
	return mbp.Balance
}

type mockEarningsProvider struct {
	Earnings pingpongEvent.EarningsDetailed
	Channels []pingpong.HermesChannel
}

// List retrieves identity's channels with all known hermeses.
func (mep *mockEarningsProvider) List(chainID int64) []pingpong.HermesChannel {
	return mep.Channels
}

// GetEarnings returns a pre-defined settlement state.
func (mep *mockEarningsProvider) GetEarningsDetailed(chainID int64, _ identity.Identity) *pingpongEvent.EarningsDetailed {
	return &mep.Earnings
}

func serviceByID(services []contract.ServiceInfoDTO, id string) (se contract.ServiceInfoDTO, found bool) {
	for i := range services {
		if services[i].ID == id {
			se = services[i]
			found = true
			return
		}
	}
	return
}

var stubLocation = market.Location{Country: "MU"}

type mockChannelAddressCalculator struct{}

func (mcac *mockChannelAddressCalculator) GetActiveChannelAddress(chainID int64, id common.Address) (common.Address, error) {
	return common.Address{}, nil
}

func (mcac *mockChannelAddressCalculator) GetActiveHermes(chainID int64) (common.Address, error) {
	return common.Address{}, nil
}

type mockProposalRepository struct {
	priceToAdd market.Price
}

func (m *mockProposalRepository) EnrichProposalWithPrice(in market.ServiceProposal) (proposal.PricedServiceProposal, error) {
	return proposal.PricedServiceProposal{
		Price:           m.priceToAdd,
		ServiceProposal: in,
	}, nil
}
