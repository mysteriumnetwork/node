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

package connection

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/trace"
)

type testContext struct {
	suite.Suite
	fakeConnectionFactory *connectionFactoryFake
	connManager           *connectionManager
	MockPaymentIssuer     *MockPaymentIssuer
	stubPublisher         *mocks.EventBus
	mockStatistics        connectionstate.Statistics
	fakeIPResolver        ip.Resolver
	fakeLocationResolver  location.OriginResolver
	config                Config
	statsReportInterval   time.Duration
	mockP2P               *mockP2PDialer
	mockTime              time.Time
	sync.RWMutex
}

var (
	consumerID            = identity.FromAddress("identity-1")
	consumerLocation      = locationstate.Location{Country: "CH"}
	activeProviderID      = identity.FromAddress("fake-node-1")
	hermesID              = common.HexToAddress("hermes")
	activeProviderContact = market.Contact{
		Type:       p2p.ContactTypeV1,
		Definition: p2p.ContactDefinition{},
	}
	activeServiceType = "fake-service"
	activeProposal    = proposal.PricedServiceProposal{
		ServiceProposal: market.ServiceProposal{
			ProviderID:  activeProviderID.Address,
			Contacts:    []market.Contact{activeProviderContact},
			ServiceType: activeServiceType,
			Location:    market.Location{},
		},
		Price: market.Price{
			PricePerHour: big.NewInt(1),
			PricePerGiB:  big.NewInt(2),
		},
	}
	activeProposalLookup = func() (proposal *proposal.PricedServiceProposal, err error) {
		return &activeProposal, nil
	}
	establishedSessionID = session.ID("session-100")
)

func (tc *testContext) SetupTest() {
	tc.Lock()
	defer tc.Unlock()

	tc.stubPublisher = mocks.NewEventBus()
	tc.mockStatistics = connectionstate.Statistics{
		BytesReceived: 10,
		BytesSent:     20,
	}
	tc.fakeConnectionFactory = &connectionFactoryFake{
		mockError: nil,
		mockConnection: &connectionMock{
			onStartReportStates: []fakeState{
				processStarted,
				connectingState,
				waitState,
				authenticatingState,
				getConfigState,
				assignIPState,
				connectedState,
			},
			onStopReportStates: []fakeState{
				exitingState,
				processExited,
			},
			onStartReportStats: tc.mockStatistics,
		},
	}

	tc.config = Config{
		IPCheck: IPCheckConfig{
			MaxAttempts:             3,
			SleepDurationAfterCheck: 1 * time.Millisecond,
		},
		KeepAlive: KeepAliveConfig{
			SendInterval:    100 * time.Millisecond,
			MaxSendErrCount: 5,
		},
	}
	tc.fakeIPResolver = ip.NewResolverMock("ip")
	tc.fakeLocationResolver = &mockLocationResolver{}
	tc.statsReportInterval = 1 * time.Millisecond

	brokerConn := nats.StartConnectionMock()
	brokerConn.MockResponse("fake-node-1.p2p-config-exchange", []byte("123"))

	tc.mockP2P = &mockP2PDialer{&mockP2PChannel{}}
	tc.mockTime = time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC)

	tc.connManager = NewManager(
		func(senderUUID string, channel p2p.Channel,
			consumer, provider identity.Identity, hermes common.Address, proposal proposal.PricedServiceProposal, price market.Price,
		) (PaymentIssuer, error) {
			tc.MockPaymentIssuer = &MockPaymentIssuer{
				stopChan: make(chan struct{}),
			}
			return tc.MockPaymentIssuer, nil
		},
		tc.fakeConnectionFactory.CreateConnection,
		tc.stubPublisher,
		tc.fakeIPResolver,
		tc.fakeLocationResolver,
		tc.config,
		tc.statsReportInterval,
		&mockValidator{},
		tc.mockP2P,
		func() {}, func() {},
	)
	tc.connManager.timeGetter = func() time.Time {
		return tc.mockTime
	}
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Exactly(tc.T(), connectionstate.Status{State: connectionstate.NotConnected}, tc.connManager.Status())
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnected() {
	tc.fakeConnectionFactory.mockError = errors.New("fatal connection error")

	assert.Error(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.NotConnected,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.Connected,
			SessionID:        establishedSessionID,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}

	go func() {
		tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	}()

	waitABit()

	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.Connecting,
			SessionID:        establishedSessionID,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)
	tc.connManager.Disconnect()
}

func (tc *testContext) TestStatusReportsNotConnected() {
	tc.fakeConnectionFactory.mockConnection.onStopReportStates = []fakeState{}
	tc.fakeConnectionFactory.mockConnection.stopBlock = make(chan struct{})
	defer func() {
		tc.fakeConnectionFactory.mockConnection.stopBlock = nil
	}()

	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), connectionstate.Connected, tc.connManager.Status().State)

	go func() {
		assert.NoError(tc.T(), tc.connManager.Disconnect())
	}()

	waitABit()
	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.Disconnecting,
			SessionID:        establishedSessionID,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)

	tc.fakeConnectionFactory.mockConnection.stopBlock <- struct{}{}

	tc.fakeConnectionFactory.mockConnection.reportState(exitingState)
	tc.fakeConnectionFactory.mockConnection.reportState(processExited)

	waitABit()
	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.NotConnected,
			SessionID:        establishedSessionID,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)
}

func (tc *testContext) TestConnectResultsInAlreadyConnectedErrorWhenConnectionExists() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
	assert.Equal(tc.T(), ErrAlreadyExists, tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
}

func (tc *testContext) TestDisconnectReturnsErrorWhenNoConnectionExists() {
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestReconnectingStatusIsReportedWhenOpenVpnGoesIntoReconnectingState() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
	tc.fakeConnectionFactory.mockConnection.reportState(reconnectingState)
	waitABit()
	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.Reconnecting,
			SessionID:        establishedSessionID,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
	assert.Equal(tc.T(), connectionstate.Connected, tc.connManager.Status().State)
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), connectionstate.NotConnected, tc.connManager.Status().State)
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
	assert.Equal(tc.T(), connectionstate.Connected, tc.connManager.Status().State)
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), connectionstate.NotConnected, tc.connManager.Status().State)

	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
	assert.Equal(tc.T(), connectionstate.Connected, tc.connManager.Status().State)
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), connectionstate.NotConnected, tc.connManager.Status().State)
}

func (tc *testContext) TestConnectFailsIfConnectionFactoryReturnsError() {
	tc.fakeConnectionFactory.mockError = errors.New("failed to create connection instance")
	assert.Error(tc.T(), tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{}))
}

func (tc *testContext) TestStatusIsConnectedWhenConnectCommandReturnsWithoutError() {
	tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.Equal(
		tc.T(),
		connectionstate.Status{
			StartedAt:        tc.mockTime,
			ConsumerID:       consumerID,
			ConsumerLocation: consumerLocation,
			HermesID:         hermesID,
			State:            connectionstate.Connected,
			SessionID:        establishedSessionID,
			Proposal:         activeProposal,
		},
		tc.connManager.Status(),
	)
}

func (tc *testContext) TestConnectingInProgressCanBeCanceled() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}
	tc.fakeConnectionFactory.mockConnection.onStopReportStates = []fakeState{}

	connectWaiter := &sync.WaitGroup{}
	connectWaiter.Add(1)
	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	}()

	waitABit()
	assert.Equal(tc.T(), connectionstate.Connecting, tc.connManager.Status().State)
	assert.NoError(tc.T(), tc.connManager.Disconnect())

	connectWaiter.Wait()

	assert.Equal(tc.T(), ErrConnectionCancelled, err)
}

func (tc *testContext) TestConnectMethodReturnsErrorIfConnectionExitsDuringConnect() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}
	tc.fakeConnectionFactory.mockConnection.onStopReportStates = []fakeState{}
	connectWaiter := sync.WaitGroup{}
	connectWaiter.Add(1)

	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	}()
	waitABit()
	tc.fakeConnectionFactory.mockConnection.reportState(processExited)
	connectWaiter.Wait()
	assert.Equal(tc.T(), ErrConnectionFailed, err)
}

func (tc *testContext) Test_PaymentManager_WhenManagerMadeConnectionIsStarted() {
	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	waitABit()
	assert.NoError(tc.T(), err)
	assert.True(tc.T(), tc.MockPaymentIssuer.StartCalled())
}

func (tc *testContext) Test_PaymentManager_OnConnectErrorIsStopped() {
	tc.fakeConnectionFactory.mockConnection.onStartReturnError = errors.New("fatal connection error")
	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.Error(tc.T(), err)
	assert.True(tc.T(), tc.MockPaymentIssuer.StopCalled())
}

func (tc *testContext) Test_SessionEndPublished_OnConnectError() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReturnError = errors.New("fatal connection error")
	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.Error(tc.T(), err)

	history := tc.stubPublisher.GetEventHistory()

	found := false

	for _, v := range history {
		if v.Topic == connectionstate.AppTopicConnectionSession {
			event := v.Event.(connectionstate.AppEventConnectionSession)
			if event.Status == connectionstate.SessionEndedStatus {
				found = true

				assert.Equal(tc.T(), connectionstate.SessionEndedStatus, event.Status)
				assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
				assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
				assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
				assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
			}
		}
	}

	assert.True(tc.T(), found)
}

func (tc *testContext) Test_ManagerPublishesEvents() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{
		connectedState,
	}

	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.NoError(tc.T(), err)

	waitABit()

	history := tc.stubPublisher.GetEventHistory()
	assert.True(tc.T(), len(history) >= 4)

	// Check if published to all expected topics.
	expectedTopics := [...]string{connectionstate.AppTopicConnectionStatistics, connectionstate.AppTopicConnectionState, connectionstate.AppTopicConnectionSession}
	for _, v := range expectedTopics {
		var published bool
		for _, h := range history {
			if v == h.Topic {
				published = true
			}
		}
		tc.Assert().Truef(published, "expected publish event to %s", v)
	}

	// Check received events data.
	for _, v := range history {
		if v.Topic == connectionstate.AppTopicConnectionStatistics {
			event := v.Event.(connectionstate.AppEventConnectionStatistics)
			assert.True(tc.T(), event.Stats.BytesReceived == tc.mockStatistics.BytesReceived)
			assert.True(tc.T(), event.Stats.BytesSent == tc.mockStatistics.BytesSent)
		}
		if v.Topic == connectionstate.AppTopicConnectionState && v.Event.(connectionstate.AppEventConnectionState).State == connectionstate.Connected {
			event := v.Event.(connectionstate.AppEventConnectionState)
			assert.Equal(tc.T(), connectionstate.Connected, event.State)
			assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
			assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
			assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
			assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
		}
		if v.Topic == connectionstate.AppTopicConnectionState && v.Event.(connectionstate.AppEventConnectionState).State == connectionstate.StateIPNotChanged {
			event := v.Event.(connectionstate.AppEventConnectionState)
			assert.Equal(tc.T(), connectionstate.StateIPNotChanged, event.State)
			assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
			assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
			assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
			assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
		}
		if v.Topic == connectionstate.AppTopicConnectionSession {
			event := v.Event.(connectionstate.AppEventConnectionSession)
			assert.Equal(tc.T(), connectionstate.SessionCreatedStatus, event.Status)
			assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
			assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
			assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
			assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
		}
	}
}

func (tc *testContext) Test_ManagerNotifiesAboutSessionIPNotChanged() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{
		connectedState,
	}

	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.NoError(tc.T(), err)

	assert.Eventually(tc.T(), func() bool {
		// Check that state event with StateIPNotChanged status was called.
		history := tc.stubPublisher.GetEventHistory()
		for _, v := range history {
			if v.Topic == connectionstate.AppTopicConnectionState && v.Event.(connectionstate.AppEventConnectionState).State == connectionstate.StateIPNotChanged {
				return true
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond)

	// Check that status sender was called with status code.
	expectedStatusMsg := &pb.SessionStatus{
		ConsumerID: consumerID.Address,
		SessionID:  string(establishedSessionID),
		Code:       uint32(connectivity.StatusSessionIPNotChanged),
		Message:    "",
	}
	assert.True(
		tc.T(),
		proto.Equal(expectedStatusMsg, tc.mockP2P.ch.getSentMsg()),
		fmt.Sprintf("Session status are not equal:\nexpected: %v\nactual:  %v", expectedStatusMsg, tc.mockP2P.ch.getSentMsg()),
	)
}

func (tc *testContext) Test_ManagerNotifiesAboutSuccessfulConnection() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{
		connectedState,
	}

	// Simulate IP change.
	tc.connManager.ipResolver = ip.NewResolverMockMultiple("127.0.0.1", "10.0.0.4", "10.0.5")

	err := tc.connManager.Connect(consumerID, hermesID, activeProposalLookup, ConnectParams{})
	assert.NoError(tc.T(), err)

	waitABit()

	// Check that state event with StateIPNotChanged status was not called.
	history := tc.stubPublisher.GetEventHistory()
	var ipNotChangedEvent *mocks.EventBusEntry
	for _, v := range history {
		if v.Topic == connectionstate.AppTopicConnectionState && v.Event.(connectionstate.AppEventConnectionState).State == connectionstate.StateIPNotChanged {
			ipNotChangedEvent = &v
		}
	}
	assert.Nil(tc.T(), ipNotChangedEvent)

	// Check that status sender was called with status code.
	expectedStatusMsg := &pb.SessionStatus{
		ConsumerID: consumerID.Address,
		SessionID:  string(establishedSessionID),
		Code:       uint32(connectivity.StatusConnectionOk),
		Message:    "",
	}
	assert.True(
		tc.T(),
		proto.Equal(expectedStatusMsg, tc.mockP2P.ch.getSentMsg()),
		fmt.Sprintf("Session status are not equal:\nexpected: %v\nactual:  %v", expectedStatusMsg, tc.mockP2P.ch.getSentMsg()),
	)
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(testContext))
}

func waitABit() {
	// usually time.Sleep call gives a chance for other goroutines to kick in
	// important when testing async code
	time.Sleep(10 * time.Millisecond)
}

type MockPaymentIssuer struct {
	startCalled bool
	stopCalled  bool
	MockError   error
	stopChan    chan struct{}
	sync.Mutex
}

func (mpm *MockPaymentIssuer) Start() error {
	mpm.Lock()
	mpm.startCalled = true
	mpm.Unlock()
	<-mpm.stopChan
	return mpm.MockError
}

func (mpm *MockPaymentIssuer) StartCalled() bool {
	mpm.Lock()
	defer mpm.Unlock()
	return mpm.startCalled
}

func (mpm *MockPaymentIssuer) StopCalled() bool {
	mpm.Lock()
	defer mpm.Unlock()
	return mpm.stopCalled
}

func (mpm *MockPaymentIssuer) Stop() {
	mpm.Lock()
	defer mpm.Unlock()
	mpm.stopCalled = true
	close(mpm.stopChan)
}

func (mpm *MockPaymentIssuer) SetSessionID(string) {
}

type mockP2PDialer struct {
	ch *mockP2PChannel
}

func (m mockP2PDialer) Dial(ctx context.Context, consumerID identity.Identity, providerID identity.Identity, serviceType string, contactDef p2p.ContactDefinition, tracer *trace.Tracer) (p2p.Channel, error) {
	return m.ch, nil
}

type mockP2PChannel struct {
	status proto.Message
	lock   sync.Mutex
}

func (m *mockP2PChannel) Conn() *net.UDPConn {
	return &net.UDPConn{}
}

func (m *mockP2PChannel) getSentMsg() proto.Message {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.status
}

func (m *mockP2PChannel) Send(_ context.Context, topic string, msg *p2p.Message) (*p2p.Message, error) {
	switch topic {
	case p2p.TopicSessionCreate:
		res := &pb.SessionResponse{
			ID: string(establishedSessionID),
		}
		return p2p.ProtoMessage(res), nil
	case p2p.TopicSessionStatus:
		m.lock.Lock()
		m.status = &pb.SessionStatus{}
		msg.UnmarshalProto(m.status)
		m.lock.Unlock()

		return nil, nil
	case p2p.TopicSessionAcknowledge:
		return nil, nil
	}

	return nil, errors.New("unexpected error")
}

func (m *mockP2PChannel) Handle(topic string, handler p2p.HandlerFunc) {
}

func (m *mockP2PChannel) Tracer() *trace.Tracer {
	return nil
}

func (m *mockP2PChannel) ServiceConn() *net.UDPConn {
	raddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	conn, _ := net.DialUDP("udp", nil, raddr)
	return conn
}

func (m *mockP2PChannel) Close() error {
	return nil
}

func (m *mockP2PChannel) ID() string {
	return fmt.Sprintf("%p", m)
}

type mockValidator struct {
	errorToReturn error
}

func (mv *mockValidator) Validate(chainID int64, consumerID identity.Identity, price market.Price) error {
	return mv.errorToReturn
}

type mockLocationResolver struct{}

func (mlr *mockLocationResolver) GetOrigin() locationstate.Location {
	return consumerLocation
}
