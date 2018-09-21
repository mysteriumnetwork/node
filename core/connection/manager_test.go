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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type fakeState string

const (
	ProcessStarted      fakeState = "ProcessStarted"
	ConnectingState     fakeState = "ConnectingState"
	ReconnectingState   fakeState = "ReconnectingState"
	WaitState           fakeState = "WaitState"
	AuthenticatingState fakeState = "AuthenticatingState"
	GetConfigState      fakeState = "GetConfigState"
	AssignIPState       fakeState = "AssignIPState"
	ConnectedState      fakeState = "ConnectedState"
	ExitingState        fakeState = "ExitingState"
	ProcessExited       fakeState = "ProcessExited"
)

type testContext struct {
	suite.Suite
	fakeConnectionFactory *connectionFactoryFake
	connManager           *connectionManager
	fakeDiscoveryClient   *server.ClientFake
	fakeDialog            *fakeDialog
	fakePromiseIssuer     *fakePromiseIssuer
	sync.RWMutex
}

type connectionFactoryFake struct {
	vpnClientCreationError error
	fakeVpnClient          *vpnClientFake
}

func (cff *connectionFactoryFake) CreateConnection(connectionParams ConnectOptions, stateChannel StateChannel) (Connection, error) {
	//each test can set this value to simulate openvpn creation error, this flag is reset BEFORE each test
	if cff.vpnClientCreationError != nil {
		return nil, cff.vpnClientCreationError
	}
	stateCallback := func(state fakeState) {
		if state == ConnectedState {
			stateChannel <- Connected
		}
		if state == ExitingState {
			stateChannel <- Disconnecting
		}
		if state == ReconnectingState {
			stateChannel <- Reconnecting
		}
		//this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if state == ProcessExited {
			close(stateChannel)
		}
	}
	cff.fakeVpnClient.StateCallback(stateCallback)
	return cff.fakeVpnClient, nil
}

var (
	myID                  = identity.FromAddress("identity-1")
	activeProviderID      = identity.FromAddress("vpn-node-1")
	activeProviderContact = dto.Contact{}
	activeProposal        = dto.ServiceProposal{
		ProviderID:       activeProviderID.Address,
		ProviderContacts: []dto.Contact{activeProviderContact},
	}
)

func (tc *testContext) SetupTest() {
	tc.Lock()
	defer tc.Unlock()

	tc.fakeDiscoveryClient = server.NewClientFake()
	tc.fakeDiscoveryClient.RegisterProposal(activeProposal, nil)

	tc.fakeDialog = &fakeDialog{}
	dialogCreator := func(consumer, provider identity.Identity, contact dto.Contact) (communication.Dialog, error) {
		tc.RLock()
		defer tc.RUnlock()
		return tc.fakeDialog, nil
	}

	tc.fakePromiseIssuer = &fakePromiseIssuer{}
	promiseIssuerFactory := func(_ identity.Identity, _ communication.Dialog) PromiseIssuer {
		return tc.fakePromiseIssuer
	}

	tc.fakeConnectionFactory = &connectionFactoryFake{
		vpnClientCreationError: nil,
		fakeVpnClient: &vpnClientFake{
			nil,
			[]fakeState{
				ProcessStarted,
				ConnectingState,
				WaitState,
				AuthenticatingState,
				GetConfigState,
				AssignIPState,
				ConnectedState,
			},
			[]fakeState{
				ExitingState,
				ProcessExited,
			},
			nil,
			sync.WaitGroup{},
			sync.RWMutex{},
		},
	}

	tc.connManager = NewManager(tc.fakeDiscoveryClient, dialogCreator, promiseIssuerFactory, tc.fakeConnectionFactory)
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Exactly(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestWithUnknownProviderConnectionIsNotMade() {
	noProposalsError := errors.New("provider has no service proposals")

	assert.Equal(tc.T(), noProposalsError, tc.connManager.Connect(myID, identity.FromAddress("unknown-node"), ConnectParams{}))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnected() {
	tc.fakeConnectionFactory.vpnClientCreationError = errors.New("fatal connection error")

	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.True(tc.T(), tc.fakeDialog.closed)
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeConnectionFactory.fakeVpnClient.onStartReportStates = []fakeState{}

	go func() {
		tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	}()

	waitABit()

	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
	tc.connManager.Disconnect()
}

func (tc *testContext) TestStatusReportsDisconnectingThenNotConnected() {
	tc.fakeConnectionFactory.fakeVpnClient.onStopReportStates = []fakeState{}
	err := tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Disconnect())
	assert.Equal(tc.T(), statusDisconnecting(), tc.connManager.Status())
	tc.fakeConnectionFactory.fakeVpnClient.reportState(ExitingState)
	tc.fakeConnectionFactory.fakeVpnClient.reportState(ProcessExited)
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestConnectResultsInAlreadyConnectedErrorWhenConnectionExists() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
	assert.Equal(tc.T(), ErrAlreadyExists, tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
}

func (tc *testContext) TestDisconnectReturnsErrorWhenNoConnectionExists() {
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestReconnectingStatusIsReportedWhenOpenVpnGoesIntoReconnectingState() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
	tc.fakeConnectionFactory.fakeVpnClient.reportState(ReconnectingState)
	waitABit()
	assert.Equal(tc.T(), statusReconnecting(), tc.connManager.Status())
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

}

func (tc *testContext) TestConnectFailsIfOpenvpnFactoryReturnsError() {
	tc.fakeConnectionFactory.vpnClientCreationError = errors.New("failed to create vpn instance")
	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectParams{}))
}

func (tc *testContext) TestStatusIsConnectedWhenConnectCommandReturnsWithoutError() {
	tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestConnectingInProgressCanBeCanceled() {
	tc.fakeConnectionFactory.fakeVpnClient.onStartReportStates = []fakeState{}
	connectWaiter := &sync.WaitGroup{}
	connectWaiter.Add(1)
	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	}()

	waitABit()
	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())

	connectWaiter.Wait()

	assert.Equal(tc.T(), ErrConnectionCancelled, err)
}

func (tc *testContext) TestConnectMethodReturnsErrorIfOpenvpnClientExitsDuringConnect() {
	tc.fakeConnectionFactory.fakeVpnClient.onStartReportStates = []fakeState{}
	tc.fakeConnectionFactory.fakeVpnClient.onStopReportStates = []fakeState{}
	connectWaiter := sync.WaitGroup{}
	connectWaiter.Add(1)

	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	}()
	waitABit()
	tc.fakeConnectionFactory.fakeVpnClient.reportState(ProcessExited)
	connectWaiter.Wait()
	assert.Equal(tc.T(), ErrConnectionFailed, err)
}

func (tc *testContext) Test_PromiseIssuer_WhenManagerMadeConnectionIsStarted() {
	err := tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	assert.NoError(tc.T(), err)
	assert.True(tc.T(), tc.fakePromiseIssuer.startCalled)
}

func (tc *testContext) Test_PromiseIssuer_OnConnectErrorIsStopped() {
	tc.fakeConnectionFactory.vpnClientCreationError = errors.New("fatal connection error")

	err := tc.connManager.Connect(myID, activeProviderID, ConnectParams{})
	assert.Error(tc.T(), err)
	assert.True(tc.T(), tc.fakePromiseIssuer.stopCalled)
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(testContext))
}

type vpnClientFake struct {
	onStartReturnError  error
	onStartReportStates []fakeState
	onStopReportStates  []fakeState
	stateCallback       func(state fakeState)
	fakeProcess         sync.WaitGroup

	sync.RWMutex
}

func (foc *vpnClientFake) Start() error {
	foc.RLock()
	defer foc.RUnlock()

	if foc.onStartReturnError != nil {
		return foc.onStartReturnError
	}

	foc.fakeProcess.Add(1)
	for _, fakeState := range foc.onStartReportStates {
		foc.reportState(fakeState)
	}
	return nil
}

func (foc *vpnClientFake) Wait() error {
	foc.fakeProcess.Wait()
	return nil
}

func (foc *vpnClientFake) Stop() {
	for _, fakeState := range foc.onStopReportStates {
		foc.reportState(fakeState)
	}
	foc.fakeProcess.Done()
}

func (foc *vpnClientFake) reportState(state fakeState) {
	foc.RLock()
	defer foc.RUnlock()

	foc.stateCallback(state)
}

func (foc *vpnClientFake) StateCallback(callback func(state fakeState)) {
	foc.Lock()
	defer foc.Unlock()

	foc.stateCallback = callback
}

type fakeDialog struct {
	peerID identity.Identity
	closed bool

	sync.RWMutex
}

func (fd *fakeDialog) PeerID() identity.Identity {
	fd.RLock()
	defer fd.RUnlock()

	return fd.peerID
}

func (fd *fakeDialog) Close() error {
	fd.Lock()
	defer fd.Unlock()

	fd.closed = true
	return nil
}

func (fd *fakeDialog) Receive(consumer communication.MessageConsumer) error {
	return nil
}
func (fd *fakeDialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}

func (fd *fakeDialog) Send(producer communication.MessageProducer) error {
	return nil
}

func (fd *fakeDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return &session.CreateResponse{
			Success: true,
			Session: session.SessionDto{
				ID:     "vpn-connection-id",
				Config: []byte("{}"),
			},
		},
		nil
}

type fakePromiseIssuer struct {
	startCalled bool
	stopCalled  bool
}

func (issuer *fakePromiseIssuer) Start(proposal dto.ServiceProposal) error {
	issuer.startCalled = true
	return nil
}

func (issuer *fakePromiseIssuer) Stop() error {
	issuer.stopCalled = true
	return nil
}

func waitABit() {
	//usually time.Sleep call gives a chance for other goroutines to kick in
	//important when testing async code
	time.Sleep(10 * time.Millisecond)
}
