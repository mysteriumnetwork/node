package connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type testContext struct {
	suite.Suite
	connManager          *connectionManager
	fakeDiscoveryClient  *server.ClientFake
	fakeOpenVpn          *fakeOpenvpnClient
	fakeStatsKeeper      *fakeSessionStatsKeeper
	openvpnCreationError error
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
	tc.fakeDiscoveryClient = server.NewClientFake()
	tc.fakeDiscoveryClient.RegisterProposal(activeProposal, nil)

	dialogCreator := func(consumer, provider identity.Identity, contact dto.Contact) (communication.Dialog, error) {
		return &fakeDialog{}, nil
	}

	tc.fakeOpenVpn = &fakeOpenvpnClient{
		nil,
		[]openvpn.State{openvpn.ConnectedState},
		[]openvpn.State{openvpn.ExitingState},
		nil,
		nil,
		nil,
	}

	tc.openvpnCreationError = nil
	fakeVpnClientFactory := func(vpnSession session.SessionDto, consumerID identity.Identity, providerID identity.Identity, callback state.Callback) (openvpn.Client, error) {
		//each test can set this value to simulate openvpn creation error, this flag is reset BEFORE each test
		if tc.openvpnCreationError != nil {
			return nil, tc.openvpnCreationError
		}

		tc.fakeOpenVpn.stateCallback = callback
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

	assert.Equal(tc.T(), noProposalsError, tc.connManager.Connect(myID, identity.FromAddress("unknown-node")))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnectedAndSessionStartIsNotMarked() {
	fatalVpnError := errors.New("fatal connection error")
	tc.fakeOpenVpn.onConnectReturnError = fatalVpnError

	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestWhenManagerMadeConnectionSessionStartIsMarked() {
	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)

	assert.True(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeOpenVpn.onStartReportStates = []openvpn.State{}
	syncChannel := make(chan error)
	go func() {
		<-syncChannel
		syncChannel <- tc.connManager.Connect(myID, activeProviderID)
	}()
	syncChannel <- nil
	select {
	case err := <-syncChannel:
		assert.Fail(tc.T(), "Connect error not expected: ", err)
	default:
		assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
	}
}

func (tc *testContext) TestStatusReportsDisconnectingThenNotConnected() {
	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())

	waiter := tc.fakeOpenVpn.holdOnClose()
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	assert.Equal(tc.T(), statusDisconnecting(), tc.connManager.Status())
	close(waiter)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.True(tc.T(), tc.fakeStatsKeeper.sessionEndMarked)
}

func (tc *testContext) TestConnectResultsInAlreadyConnectedErrorWhenConnectionExists() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), ErrAlreadyExists, tc.connManager.Connect(myID, activeProviderID))
}

func (tc *testContext) TestDisconnectReturnsErrorWhenNoConnectionExists() {
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestReconnectingStatusIsReportedWhenOpenVpnGoesIntoReconnectingState() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	tc.fakeOpenVpn.reportState(openvpn.ReconnectingState)
	assert.Equal(tc.T(), statusReconnecting(), tc.connManager.Status())
}

func (tc *testContext) TestConnectedStatusIsReportedWhenOpenvpnReportsExiting() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	tc.fakeOpenVpn.reportState(openvpn.ExitingState)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	time.Sleep(10 * time.Millisecond)
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))

	assert.NoError(tc.T(), tc.connManager.Disconnect())
	time.Sleep(100 * time.Millisecond)
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))

	assert.NoError(tc.T(), tc.connManager.Disconnect())
	time.Sleep(100 * time.Millisecond)
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

}

func (tc *testContext) TestConnectFailsIfOpenvpnFactoryReturnsError() {
	tc.openvpnCreationError = errors.New("failed to create vpn instanse")
	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID))
}

func (tc *testContext) TestStatusIsConnectedWhenConnectCommandReturnsWithoutError() {
	tc.connManager.Connect(myID, activeProviderID)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
}

func (tc *testContext) TestConnectingInProgressCanBeCanceled() {
	tc.fakeOpenVpn.onStartReportStates = []openvpn.State{}
	syncChannel := make(chan error)
	go func() {
		<-syncChannel
		syncChannel <- tc.connManager.Connect(myID, activeProviderID)
	}()
	syncChannel <- nil
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	time.Sleep(100 * time.Millisecond)
	select {
	case err := <-syncChannel:
		assert.Equal(tc.T(), ErrConnectionCancelled, err)
	default:
		assert.Fail(tc.T(), "Expected error returned by connect")
	}
}

func (tc *testContext) TestConnectMethodReturnsErrorIfOpenvpnClientExitsDuringConnect() {
	tc.fakeOpenVpn.onStartReportStates = []openvpn.State{}
	tc.fakeOpenVpn.onStopReportStates = []openvpn.State{}
	syncChannel := make(chan error)

	go func() {
		<-syncChannel
		syncChannel <- tc.connManager.Connect(myID, activeProviderID)
	}()
	syncChannel <- nil
	time.Sleep(50 * time.Millisecond)
	tc.fakeOpenVpn.Stop()
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-syncChannel:
		assert.Equal(tc.T(), ErrOpenvpnProcessDied, err)
	default:
		assert.Fail(tc.T(), "Expected error returned by connect")
	}
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(testContext))
}

type fakeOpenvpnClient struct {
	onConnectReturnError error
	onStartReportStates  []openvpn.State
	onStopReportStates   []openvpn.State
	stateCallback        state.Callback
	stopProcess          chan int
	closeHoldChannel     chan int
}

func (foc *fakeOpenvpnClient) Start() error {
	foc.stopProcess = make(chan int)
	for _, openvpnState := range foc.onStartReportStates {
		foc.reportState(openvpnState)
	}
	return foc.onConnectReturnError
}

func (foc *fakeOpenvpnClient) Wait() error {
	<-foc.stopProcess
	return nil
}

func (foc *fakeOpenvpnClient) Stop() error {
	if foc.closeHoldChannel != nil {
		<-foc.closeHoldChannel
	}
	for _, openvpnState := range foc.onStopReportStates {
		foc.reportState(openvpnState)
	}
	close(foc.stopProcess)
	return nil
}

func (foc *fakeOpenvpnClient) reportState(state openvpn.State) {
	foc.stateCallback(state)
	time.Sleep(time.Millisecond)
}
func (foc *fakeOpenvpnClient) holdOnClose() chan int {
	foc.closeHoldChannel = make(chan int)
	return foc.closeHoldChannel
}

type fakeDialog struct {
	peerId identity.Identity
}

func (fd *fakeDialog) PeerID() identity.Identity {
	return fd.peerId
}

func (fd *fakeDialog) Close() error {
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
				Config: "vpn-connection-config",
			},
		},
		nil
}

type fakeSessionStatsKeeper struct {
	sessionStartMarked, sessionEndMarked bool
}

func (fsk *fakeSessionStatsKeeper) Save(stats bytescount.SessionStats) {
}

func (fsk *fakeSessionStatsKeeper) Retrieve() bytescount.SessionStats {
	return bytescount.SessionStats{}
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
