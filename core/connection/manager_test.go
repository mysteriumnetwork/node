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

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type testContext struct {
	suite.Suite
	connManager          *connectionManager
	fakeDiscoveryClient  *server.ClientFake
	fakeOpenVpn          *fakeOpenvpnClient
	fakeStatsKeeper      *fakeSessionStatsKeeper
	fakeDialog           *fakeDialog
	openvpnCreationError error

	sync.RWMutex
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

	tc.fakeOpenVpn = &fakeOpenvpnClient{
		nil,
		[]openvpn.State{
			openvpn.ProcessStarted,
			openvpn.ConnectingState,
			openvpn.WaitState,
			openvpn.AuthenticatingState,
			openvpn.GetConfigState,
			openvpn.AssignIpState,
			openvpn.ConnectedState,
		},
		[]openvpn.State{
			openvpn.ExitingState,
			openvpn.ProcessExited,
		},
		nil,
		sync.WaitGroup{},
		sync.RWMutex{},
	}

	tc.openvpnCreationError = nil
	fakeVpnClientFactory := func(vpnSession *session.Session, consumerID identity.Identity, providerID identity.Identity, callback state.Callback, options ConnectOptions) (openvpn.Process, error) {
		tc.RLock()
		defer tc.RUnlock()
		//each test can set this value to simulate openvpn creation error, this flag is reset BEFORE each test
		if tc.openvpnCreationError != nil {
			return nil, tc.openvpnCreationError
		}

		tc.fakeOpenVpn.StateCallback(callback)
		return tc.fakeOpenVpn, nil
	}
	tc.fakeStatsKeeper = &fakeSessionStatsKeeper{}

	tc.connManager = NewManager(tc.fakeDiscoveryClient, dialogCreator, fakeVpnClientFactory, tc.fakeStatsKeeper)
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Exactly(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestWithUnknownProviderConnectionIsNotMade() {
	noProposalsError := errors.New("provider has no service proposals")

	assert.Equal(tc.T(), noProposalsError, tc.connManager.Connect(myID, identity.FromAddress("unknown-node"), ConnectOptions{}))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnectedAndSessionStartIsNotMarked() {
	fatalVpnError := errors.New("fatal connection error")
	tc.fakeOpenVpn.onStartReturnError = fatalVpnError

	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.True(tc.T(), tc.fakeDialog.closed)
	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestWhenManagerMadeConnectionSessionStartIsMarked() {
	err := tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
	assert.NoError(tc.T(), err)

	assert.True(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeOpenVpn.onStartReportStates = []openvpn.State{}

	go func() {
		tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
		assert.Fail(tc.T(), "This should never return")
	}()

	waitABit()

	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
}

func (tc *testContext) TestStatusReportsDisconnectingThenNotConnected() {
	tc.fakeOpenVpn.onStopReportStates = []openvpn.State{}
	err := tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Disconnect())
	assert.Equal(tc.T(), statusDisconnecting(), tc.connManager.Status())
	tc.fakeOpenVpn.reportState(openvpn.ExitingState)
	tc.fakeOpenVpn.reportState(openvpn.ProcessExited)
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.True(tc.T(), tc.fakeStatsKeeper.sessionEndMarked)
}

func (tc *testContext) TestConnectResultsInAlreadyConnectedErrorWhenConnectionExists() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
	assert.Equal(tc.T(), ErrAlreadyExists, tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
}

func (tc *testContext) TestDisconnectReturnsErrorWhenNoConnectionExists() {
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestReconnectingStatusIsReportedWhenOpenVpnGoesIntoReconnectingState() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
	tc.fakeOpenVpn.reportState(openvpn.ReconnectingState)
	waitABit()
	assert.Equal(tc.T(), statusReconnecting(), tc.connManager.Status())
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

}

func (tc *testContext) TestConnectFailsIfOpenvpnFactoryReturnsError() {
	tc.openvpnCreationError = errors.New("failed to create vpn instance")
	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID, ConnectOptions{}))
}

func (tc *testContext) TestStatusIsConnectedWhenConnectCommandReturnsWithoutError() {
	tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestConnectingInProgressCanBeCanceled() {
	tc.fakeOpenVpn.onStartReportStates = []openvpn.State{}
	connectWaiter := &sync.WaitGroup{}
	connectWaiter.Add(1)
	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
	}()

	waitABit()
	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())

	connectWaiter.Wait()

	assert.Equal(tc.T(), ErrConnectionCancelled, err)
}

func (tc *testContext) TestConnectMethodReturnsErrorIfOpenvpnClientExitsDuringConnect() {
	tc.fakeOpenVpn.onStartReportStates = []openvpn.State{}
	tc.fakeOpenVpn.onStopReportStates = []openvpn.State{}
	connectWaiter := sync.WaitGroup{}
	connectWaiter.Add(1)

	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(myID, activeProviderID, ConnectOptions{})
	}()
	waitABit()
	tc.fakeOpenVpn.reportState(openvpn.ProcessExited)
	connectWaiter.Wait()
	assert.Equal(tc.T(), ErrOpenvpnProcessDied, err)
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(testContext))
}

type fakeOpenvpnClient struct {
	onStartReturnError  error
	onStartReportStates []openvpn.State
	onStopReportStates  []openvpn.State
	stateCallback       state.Callback
	fakeProcess         sync.WaitGroup

	sync.RWMutex
}

func (foc *fakeOpenvpnClient) Start() error {
	foc.RLock()
	defer foc.RUnlock()

	if foc.onStartReturnError != nil {
		return foc.onStartReturnError
	}

	foc.fakeProcess.Add(1)
	for _, openvpnState := range foc.onStartReportStates {
		foc.reportState(openvpnState)
	}
	return nil
}

func (foc *fakeOpenvpnClient) Wait() error {
	foc.fakeProcess.Wait()
	return nil
}

func (foc *fakeOpenvpnClient) Stop() {
	for _, openvpnState := range foc.onStopReportStates {
		foc.reportState(openvpnState)
	}
	foc.fakeProcess.Done()
}

func (foc *fakeOpenvpnClient) reportState(state openvpn.State) {
	foc.RLock()
	defer foc.RUnlock()

	foc.stateCallback(state)
}

func (foc *fakeOpenvpnClient) StateCallback(callback state.Callback) {
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
	return &session.SessionCreateResponse{
			Success: true,
			Message: "Everything is great!",
			Session: session.SessionDto{
				ID:     "vpn-connection-id",
				Config: []byte("{}"),
			},
		},
		nil
}

type fakeSessionStatsKeeper struct {
	sessionStartMarked, sessionEndMarked bool
}

func (fsk *fakeSessionStatsKeeper) Save(stats stats.SessionStats) {
}

func (fsk *fakeSessionStatsKeeper) Retrieve() stats.SessionStats {
	return stats.SessionStats{}
}

func (fsk *fakeSessionStatsKeeper) MarkSessionStart() {
	fsk.sessionStartMarked = true
}

func (fsk *fakeSessionStatsKeeper) GetSessionDuration() time.Duration {
	return time.Duration(0)
}

func (fsk *fakeSessionStatsKeeper) MarkSessionEnd() {
	fsk.sessionEndMarked = true
}

func waitABit() {
	//usually time.Sleep call gives a chance for other goroutines to kick in
	//important when testing async code
	time.Sleep(10 * time.Millisecond)
}
