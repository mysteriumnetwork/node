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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type testContext struct {
	suite.Suite
	fakeConnectionFactory *connectionFactoryFake
	connManager           *connectionManager
	fakeDialog            *fakeDialog
	fakePromiseIssuer     *fakePromiseIssuer
	stubPublisher         *StubPublisher
	mockStatistics        consumer.SessionStatistics
	sync.RWMutex
}

var (
	consumerID            = identity.FromAddress("identity-1")
	activeProviderID      = identity.FromAddress("fake-node-1")
	activeProviderContact = market.Contact{}
	activeServiceType     = "fake-service"
	activeProposal        = market.ServiceProposal{
		ProviderID:        activeProviderID.Address,
		ProviderContacts:  []market.Contact{activeProviderContact},
		ServiceType:       activeServiceType,
		ServiceDefinition: &fakeServiceDefinition{},
	}
	establishedSessionID = session.ID("session-100")
)

func (tc *testContext) SetupTest() {
	tc.Lock()
	defer tc.Unlock()

	tc.stubPublisher = NewStubPublisher()
	tc.fakeDialog = &fakeDialog{sessionID: establishedSessionID}
	dialogCreator := func(consumer, provider identity.Identity, contact market.Contact) (communication.Dialog, error) {
		tc.RLock()
		defer tc.RUnlock()
		return tc.fakeDialog, nil
	}

	tc.fakePromiseIssuer = &fakePromiseIssuer{}
	promiseIssuerFactory := func(_ identity.Identity, _ communication.Dialog) PromiseIssuer {
		return tc.fakePromiseIssuer
	}
	tc.mockStatistics = consumer.SessionStatistics{
		BytesReceived: 10,
		BytesSent:     20,
	}
	tc.fakeConnectionFactory = &connectionFactoryFake{
		mockError: nil,
		mockConnection: &connectionFake{
			nil,
			[]fakeState{
				processStarted,
				connectingState,
				waitState,
				authenticatingState,
				getConfigState,
				assignIPState,
				connectedState,
			},
			[]fakeState{
				exitingState,
				processExited,
			},
			nil,
			tc.mockStatistics,
			sync.WaitGroup{},
			sync.RWMutex{},
		},
	}

	tc.connManager = NewManager(
		dialogCreator,
		promiseIssuerFactory,
		tc.fakeConnectionFactory.CreateConnection,
		tc.stubPublisher,
	)
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Exactly(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnected() {
	tc.fakeConnectionFactory.mockError = errors.New("fatal connection error")

	assert.Error(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.True(tc.T(), tc.fakeDialog.closed)
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected(establishedSessionID), tc.connManager.Status())
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}

	go func() {
		tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	}()

	waitABit()

	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
	tc.connManager.Disconnect()
}

func (tc *testContext) TestStatusReportsDisconnectingThenNotConnected() {
	tc.fakeConnectionFactory.mockConnection.onStopReportStates = []fakeState{}
	err := tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected(establishedSessionID), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Disconnect())
	assert.Equal(tc.T(), statusDisconnecting(), tc.connManager.Status())
	tc.fakeConnectionFactory.mockConnection.reportState(exitingState)
	tc.fakeConnectionFactory.mockConnection.reportState(processExited)
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestConnectResultsInAlreadyConnectedErrorWhenConnectionExists() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), ErrAlreadyExists, tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
}

func (tc *testContext) TestDisconnectReturnsErrorWhenNoConnectionExists() {
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestReconnectingStatusIsReportedWhenOpenVpnGoesIntoReconnectingState() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
	tc.fakeConnectionFactory.mockConnection.reportState(reconnectingState)
	waitABit()
	assert.Equal(tc.T(), statusReconnecting(), tc.connManager.Status())
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected(establishedSessionID), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected(establishedSessionID), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected(establishedSessionID), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

}

func (tc *testContext) TestConnectFailsIfConnectionFactoryReturnsError() {
	tc.fakeConnectionFactory.mockError = errors.New("failed to create connection instance")
	assert.Error(tc.T(), tc.connManager.Connect(consumerID, activeProposal, ConnectParams{}))
}

func (tc *testContext) TestStatusIsConnectedWhenConnectCommandReturnsWithoutError() {
	tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	assert.Equal(tc.T(), statusConnected(establishedSessionID), tc.connManager.Status())
}

func (tc *testContext) TestConnectingInProgressCanBeCanceled() {
	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{}
	connectWaiter := &sync.WaitGroup{}
	connectWaiter.Add(1)
	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
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
		err = tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	}()
	waitABit()
	tc.fakeConnectionFactory.mockConnection.reportState(processExited)
	connectWaiter.Wait()
	assert.Equal(tc.T(), ErrConnectionFailed, err)
}

func (tc *testContext) Test_PromiseIssuer_WhenManagerMadeConnectionIsStarted() {
	err := tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.True(tc.T(), tc.fakePromiseIssuer.startCalled)
}

func (tc *testContext) Test_PromiseIssuer_OnConnectErrorIsStopped() {
	tc.fakeConnectionFactory.mockConnection.onStartReturnError = errors.New("fatal connection error")
	err := tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	assert.Error(tc.T(), err)
	assert.True(tc.T(), tc.fakePromiseIssuer.stopCalled)
}

func (tc *testContext) Test_ManagerPublishesEvents() {
	tc.stubPublisher.Clear()

	tc.fakeConnectionFactory.mockConnection.onStartReportStates = []fakeState{
		connectedState,
	}

	err := tc.connManager.Connect(consumerID, activeProposal, ConnectParams{})
	assert.NoError(tc.T(), err)

	waitABit()

	history := tc.stubPublisher.GetEventHistory()
	assert.Len(tc.T(), history, 2)

	for _, v := range history {
		if v.calledWithTopic == StatisticsEventTopic {
			event := v.calledWithArgs[0].(consumer.SessionStatistics)
			assert.True(tc.T(), event.BytesReceived == tc.mockStatistics.BytesReceived)
			assert.True(tc.T(), event.BytesSent == tc.mockStatistics.BytesSent)
		}
		if v.calledWithTopic == StateEventTopic {
			event := v.calledWithArgs[0].(StateEvent)
			assert.Equal(tc.T(), Connected, event.State)
			assert.Equal(tc.T(), consumerID, event.SessionInfo.ConsumerID)
			assert.Equal(tc.T(), establishedSessionID, event.SessionInfo.SessionID)
			assert.Equal(tc.T(), activeProposal.ProviderID, event.SessionInfo.Proposal.ProviderID)
			assert.Equal(tc.T(), activeProposal.ServiceType, event.SessionInfo.Proposal.ServiceType)
		}
	}
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
