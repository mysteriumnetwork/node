package connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type testContext struct {
	suite.Suite
	connManager          *connectionManager
	fakeDiscoveryClient  *server.ClientFake
	fakeOpenVpn          *fakeOpenvpnClient
	fakeStatsKeeper      *fakeSessionStatsKeeper
	fakeDialog           *fakeDialog
	openvpnCreationError error
	locationDetector     *fakeLocationDetector
	locationOriginal     location.Cache
}

var (
	myID                  = identity.FromAddress("identity-1")
	myLocationUnknown     = location.Location{}
	myLocationLT          = location.Location{"8.8.8.1", "LT"}
	myLocationLV          = location.Location{"8.8.8.2", "LV"}
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
	tc.fakeDialog = &fakeDialog{}

	dialogCreator := func(consumer, provider identity.Identity, contact dto.Contact) (communication.Dialog, error) {
		return tc.fakeDialog, nil
	}

	tc.fakeOpenVpn = &fakeOpenvpnClient{
		nil,
		[]openvpn.State{
			openvpn.ConnectingState,
			openvpn.WaitState,
			openvpn.AuthenticatingState,
			openvpn.GetConfigState,
			openvpn.AssignIpState,
			openvpn.ConnectedState,
		},
		[]openvpn.State{
			openvpn.ExitingState,
		},
		nil,
		sync.WaitGroup{},
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
	tc.locationDetector = &fakeLocationDetector{}
	tc.locationOriginal = location.NewLocationCache()

	tc.connManager = NewManager(tc.fakeDiscoveryClient, dialogCreator, fakeVpnClientFactory, tc.fakeStatsKeeper, tc.locationDetector, tc.locationOriginal)
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Exactly(tc.T(), statusNotConnected(), tc.connManager.Status())
}

func (tc *testContext) TestConnectDetectsOriginalLocation() {
	tc.locationOriginal.Set(myLocationUnknown)
	tc.locationDetector.location = myLocationLT

	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)

	assert.Equal(tc.T(), myLocationLT, tc.locationOriginal.Get())
}

func (tc *testContext) TestWhenArrivingToLatviaConnectDetectsNewLocation() {
	tc.locationOriginal.Set(myLocationLT)
	tc.locationDetector.location = myLocationLV

	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), myLocationLV, tc.locationOriginal.Get())
}

func (tc *testContext) TestConnectsSucceedsWhenDetectsOriginalLocationFails() {
	tc.locationOriginal.Set(myLocationLT)
	tc.locationDetector.error = errors.New("failed to detect location")

	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)

	assert.Equal(tc.T(), myLocationUnknown, tc.locationOriginal.Get())
}

func (tc *testContext) TestWithUnknownProviderConnectionIsNotMade() {
	noProposalsError := errors.New("provider has no service proposals")

	assert.Equal(tc.T(), noProposalsError, tc.connManager.Connect(myID, identity.FromAddress("unknown-node")))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnectedAndSessionStartIsNotMarked() {
	fatalVpnError := errors.New("fatal connection error")
	tc.fakeOpenVpn.onStartReturnError = fatalVpnError

	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.True(tc.T(), tc.fakeDialog.closed)
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

	go func() {
		tc.connManager.Connect(myID, activeProviderID)
		assert.Fail(tc.T(), "This should never return")
	}()

	waitABit()

	assert.Equal(tc.T(), statusConnecting(), tc.connManager.Status())
}

func (tc *testContext) TestStatusReportsDisconnectingThenNotConnected() {
	tc.fakeOpenVpn.onStopReportStates = []openvpn.State{}
	err := tc.connManager.Connect(myID, activeProviderID)
	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Disconnect())
	assert.Equal(tc.T(), statusDisconnecting(), tc.connManager.Status())
	tc.fakeOpenVpn.reportState(openvpn.ExitingState)
	waitABit()
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
	waitABit()
	assert.Equal(tc.T(), statusReconnecting(), tc.connManager.Status())
}

func (tc *testContext) TestDoubleDisconnectResultsInError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())
	assert.Equal(tc.T(), ErrNoConnection, tc.connManager.Disconnect())
}

func (tc *testContext) TestTwoConnectDisconnectCyclesReturnNoError() {
	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
	assert.Equal(tc.T(), statusNotConnected(), tc.connManager.Status())

	assert.NoError(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), statusConnected("vpn-connection-id"), tc.connManager.Status())
	assert.NoError(tc.T(), tc.connManager.Disconnect())
	waitABit()
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
	connectWaiter := &sync.WaitGroup{}
	connectWaiter.Add(1)
	var err error
	go func() {
		defer connectWaiter.Done()
		err = tc.connManager.Connect(myID, activeProviderID)
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
		err = tc.connManager.Connect(myID, activeProviderID)
	}()
	waitABit()
	tc.fakeOpenVpn.reportState(openvpn.ExitingState)
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
}

func (foc *fakeOpenvpnClient) Start() error {
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

func (foc *fakeOpenvpnClient) Stop() error {
	for _, openvpnState := range foc.onStopReportStates {
		foc.reportState(openvpnState)
	}
	foc.fakeProcess.Done()
	return nil
}

func (foc *fakeOpenvpnClient) reportState(state openvpn.State) {
	foc.stateCallback(state)
}

type fakeDialog struct {
	peerId identity.Identity
	closed bool
}

func (fd *fakeDialog) PeerID() identity.Identity {
	return fd.peerId
}

func (fd *fakeDialog) Close() error {
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
				Config: []byte{},
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

func waitABit() {
	//usually time.Sleep call gives a chance for other goroutines to kick in
	//important when testing async code
	time.Sleep(10 * time.Millisecond)
}

type fakeLocationDetector struct {
	location location.Location
	error    error
}

// Maps current ip to country
func (d *fakeLocationDetector) DetectLocation() (location.Location, error) {
	return d.location, d.error
}
