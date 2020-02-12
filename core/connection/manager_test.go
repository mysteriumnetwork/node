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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type testContext struct {
	suite.Suite
	fakeConnectionFactory *connectionFactoryFake
	connManager           *connectionManager
	mockDialog            *mockDialog
	MockPaymentIssuer     *MockPaymentIssuer
	stubPublisher         *StubPublisher
	mockStatistics        consumer.SessionStatistics
	fakeResolver          ip.Resolver
	ipCheckParams         IPCheckParams
	statusSender          *mockStatusSender
	sync.RWMutex
}

var (
	consumerID            = identity.FromAddress("identity-1")
	activeProviderID      = identity.FromAddress("fake-node-1")
	accountantID          = identity.FromAddress("accountant")
	activeProviderContact = market.Contact{}
	activeServiceType     = "fake-service"
	activeProposal        = market.ServiceProposal{
		ProviderID:        activeProviderID.Address,
		ProviderContacts:  []market.Contact{activeProviderContact},
		ServiceType:       activeServiceType,
		ServiceDefinition: &fakeServiceDefinition{},
	}
	establishedSessionID = session.ID("session-100")
	paymentInfo          *promise.PaymentInfo
)

func (tc *testContext) SetupTest() {
	tc.Lock()
	defer tc.Unlock()

	tc.stubPublisher = NewStubPublisher()
	dialogCreator := func(consumer, provider identity.Identity, contact market.Contact) (communication.Dialog, error) {
		tc.Lock()
		defer tc.Unlock()
		tc.mockDialog = &mockDialog{
			sessionID:   establishedSessionID,
			paymentInfo: paymentInfo,
		}
		return tc.mockDialog, nil
	}

	tc.mockStatistics = consumer.SessionStatistics{
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

	tc.ipCheckParams = IPCheckParams{
		MaxAttempts:             3,
		SleepDurationAfterCheck: 1 * time.Millisecond,
		Done:                    make(chan struct{}, 1),
	}

	tc.statusSender = &mockStatusSender{}
	tc.fakeResolver = ip.NewResolverMock("ip")

	tc.connManager = NewManager(
		dialogCreator,
		func(paymentInfo *promise.PaymentInfo,
			dialog communication.Dialog,
			consumer, provider, accountant identity.Identity, proposal market.ServiceProposal, sessionID string) (PaymentIssuer, error) {
			if paymentInfo == nil {
				paymentInfo = &promise.PaymentInfo{}
			}
			tc.MockPaymentIssuer = &MockPaymentIssuer{
				initialState:      *paymentInfo,
				paymentDefinition: dto.PaymentRate{},
				stopChan:          make(chan struct{}),
			}
			return tc.MockPaymentIssuer, nil
		},
		tc.fakeConnectionFactory.CreateConnection,
		tc.stubPublisher,
		tc.statusSender,
		tc.fakeResolver,
		tc.ipCheckParams,
		false,
	)
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Exactly(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnected() {
	tc.fakeConnectionFactory.mockError = errors.New("fatal connection error")

	assert.Error(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected(establishedSessionID, activeProposal), tc.connManager.Status())
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}

	go func() {
		tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	}()

	waitABit()

	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
	tc.connManager.Disconnect()
}

func (tc *testContext) TestStatusReportsNotConnected() {
	tc.fakeConnectionFactory.mockConnection.onStopReportStates = []fakeState{}
	tc.fakeConnectionFactory.mockConnection.stopBlock = make(chan struct{})
	defer func() {
		tc.fakeConnectionFactory.mockConnection.stopBlock = nil
	}()

	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected(establishedSessionID, activeProposal), tc.connManager.Status())

	go func() {
		assert.NoError(tc.T(), tc.connManager.Disconnect())
	}()

	waitABit()
	assert.Equal(tc.T(), statusDisconnecting(), tc.connManager.Status())

	tc.fakeConnectionFactory.mockConnection.stopBlock <- struct{}{}

	tc.fakeConnectionFactory.mockConnection.reportState(exitingState)
	tc.fakeConnectionFactory.mockConnection.reportState(processExited)

	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestConnectResultsInAlreadyConnectedErrorWhenConnectionExists() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), ErrAlreadyExists, tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
}

func (tc *testContext) TestDisconnectReturnsErrorWhenNoConnectionExists() {
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestReconnectingStatusIsReportedWhenOpenVpnGoesIntoReconnectingState() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
	tc.fakeConnectionFactory.mockConnection.reportState(reconnectingState)
	waitABit()
	assert.Equal(tc.T(), statusReconnecting(), tc.connManager.Status())
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected(establishedSessionID, activeProposal), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected(establishedSessionID, activeProposal), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected(establishedSessionID, activeProposal), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestConnectFailsIfConnectionFactoryReturnsError() {
	tc.fakeConnectionFactory.mockError = errors.New("failed to create connection instance")
	assert.Error(tc.T(), tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{}))
}

func (tc *testContext) TestStatusIsConnectedWhenConnectCommandReturnsWithoutError() {
	tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.Equal(tc.T(), statusConnected(establishedSessionID, activeProposal), tc.connManager.Status())
}

func (tc *testContext) TestConnectingInProgressCanBeCanceled() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}
	tc.fakeConnectionFactory.mockConnection.onStopReportStates = []fakeState{}

	connectWaiter := &sync.WaitGroup{}
	connectWaiter.Add(1)
	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	}()

	waitABit()
	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
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
		err = tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	}()
	waitABit()
	tc.fakeConnectionFactory.mockConnection.reportState(processExited)
	connectWaiter.Wait()
	assert.Equal(tc.T(), ErrConnectionFailed, err)
}

func (tc *testContext) Test_PaymentManager_WhenManagerMadeConnectionIsStarted() {
	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	waitABit()
	assert.NoError(tc.T(), err)
	assert.True(tc.T(), tc.MockPaymentIssuer.StartCalled())
}

func (tc *testContext) Test_PaymentManager_OnConnectErrorIsStopped() {
	tc.fakeConnectionFactory.mockConnection.onStartReturnError = errors.New("fatal connection error")
	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.Error(tc.T(), err)
	assert.True(tc.T(), tc.MockPaymentIssuer.StopCalled())
}

func (tc *testContext) Test_SessionEndPublished_OnConnectError() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReturnError = errors.New("fatal connection error")
	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.Error(tc.T(), err)

	history := tc.stubPublisher.GetEventHistory()

	found := false

	for _, v := range history {
		if v.calledWithTopic == AppTopicConsumerSession {
			event := v.calledWithData.(SessionEvent)
			if event.Status == SessionEndedStatus {
				found = true

				assert.Equal(tc.T(), SessionEndedStatus, event.Status)
				assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
				assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
				assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
				assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
			}
		}
	}

	assert.True(tc.T(), found)
}

func (tc *testContext) Test_ManagerSetsPaymentInfo() {
	defer func() {
		paymentInfo = nil
	}()
	paymentInfo = &promise.PaymentInfo{
		LastPromise: promise.LastPromise{
			SequenceID: 1,
			Amount:     200,
		},
		FreeCredit: 100,
	}
	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.Nil(tc.T(), err)
	assert.Exactly(tc.T(), *paymentInfo, tc.MockPaymentIssuer.initialState)
}

func (tc *testContext) Test_ManagerPublishesEvents() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{
		connectedState,
	}

	err := tc.connManager.Connect(consumerID, accountantID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)

	waitABit()

	history := tc.stubPublisher.GetEventHistory()
	assert.Len(tc.T(), history, 4)

	for _, v := range history {
		if v.calledWithTopic == AppTopicConsumerStatistics {
			event := v.calledWithData.(SessionStatsEvent)
			assert.True(tc.T(), event.Stats.BytesReceived == tc.mockStatistics.BytesReceived)
			assert.True(tc.T(), event.Stats.BytesSent == tc.mockStatistics.BytesSent)
		}
		if v.calledWithTopic == AppTopicConsumerConnectionState && v.calledWithData.(StateEvent).State == Connected {
			event := v.calledWithData.(StateEvent)
			assert.Equal(tc.T(), Connected, event.State)
			assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
			assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
			assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
			assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
		}
		if v.calledWithTopic == AppTopicConsumerConnectionState && v.calledWithData.(StateEvent).State == StateIPNotChanged {
			event := v.calledWithData.(StateEvent)
			assert.Equal(tc.T(), StateIPNotChanged, event.State)
			assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
			assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
			assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
			assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
		}
		if v.calledWithTopic == AppTopicConsumerSession {
			event := v.calledWithData.(SessionEvent)
			assert.Equal(tc.T(), SessionCreatedStatus, event.Status)
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

	err := tc.connManager.Connect(consumerID, consumerID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)

	waitABit()

	// Check that state event with StateIPNotChanged status was called.
	history := tc.stubPublisher.GetEventHistory()
	var ipNotChangedEvent *StubPublisherEvent
	for _, v := range history {
		if v.calledWithTopic == AppTopicConsumerConnectionState && v.calledWithData.(StateEvent).State == StateIPNotChanged {
			ipNotChangedEvent = &v
		}
	}
	assert.NotNil(tc.T(), ipNotChangedEvent)

	// Check that status sender was called with status code.
	expectedStatusMsg := connectivity.StatusMessage{
		SessionID:  string(establishedSessionID),
		StatusCode: connectivity.StatusSessionIPNotChanged,
		Message:    "",
	}
	assert.NotNil(tc.T(), tc.statusSender.sentMsg)
	assert.Equal(tc.T(), &expectedStatusMsg, tc.statusSender.sentMsg)
}

func (tc *testContext) Test_ManagerNotifiesAboutSuccessfulConnection() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{
		connectedState,
	}

	// Simulate IP change.
	tc.connManager.ipResolver = ip.NewResolverMock("10.0.0.4", "10.0.5")

	err := tc.connManager.Connect(consumerID, consumerID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)

	waitABit()

	// Check that state event with StateIPNotChanged status was not called.
	history := tc.stubPublisher.GetEventHistory()
	var ipNotChangedEvent *StubPublisherEvent
	for _, v := range history {
		if v.calledWithTopic == AppTopicConsumerConnectionState && v.calledWithData.(StateEvent).State == StateIPNotChanged {
			ipNotChangedEvent = &v
		}
	}
	assert.Nil(tc.T(), ipNotChangedEvent)

	// Check that status sender was called with status code.
	expectedStatusMsg := connectivity.StatusMessage{
		SessionID:  string(establishedSessionID),
		StatusCode: connectivity.StatusConnectionOk,
		Message:    "",
	}

	assert.Equal(tc.T(), expectedStatusMsg, tc.statusSender.getSentMsg())
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(testContext))
}

func waitABit() {
	//usually time.Sleep call gives a chance for other goroutines to kick in
	//important when testing async code
	time.Sleep(10 * time.Millisecond)
}

type fakeServiceDefinition struct{}

func (fs *fakeServiceDefinition) GetLocation() market.Location { return market.Location{} }

type MockPaymentIssuer struct {
	initialState      promise.PaymentInfo
	paymentDefinition dto.PaymentRate
	startCalled       bool
	stopCalled        bool
	MockError         error
	stopChan          chan struct{}
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

type mockStatusSender struct {
	sentMsg *connectivity.StatusMessage
	sync.Mutex
}

func (s *mockStatusSender) Send(dialog communication.Sender, msg *connectivity.StatusMessage) {
	s.Lock()
	defer s.Unlock()
	s.sentMsg = msg
}

func (s *mockStatusSender) getSentMsg() connectivity.StatusMessage {
	s.Lock()
	defer s.Unlock()
	return *s.sentMsg
}
