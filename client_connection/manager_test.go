package client_connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type testContext struct {
	suite.Suite
	connManager         *connectionManager
	fakeDiscoveryClient *server.ClientFake
	fakeOpenVpn         *fakeOpenvpnClient
	fakeStatsKeeper     *fakeSessionStatsKeeper
}

var (
	myID                  = identity.FromAddress("identity-1")
	activeProviderID      = identity.FromAddress("vpn-node-1")
	activeProviderContact = dto_discovery.Contact{}
	activeProposal        = dto_discovery.ServiceProposal{
		ProviderID:       activeProviderID.Address,
		ProviderContacts: []dto_discovery.Contact{activeProviderContact},
	}
)

func (tc *testContext) SetupTest() {
	tc.fakeDiscoveryClient = server.NewClientFake()
	tc.fakeDiscoveryClient.RegisterProposal(activeProposal, nil)

	dialogEstablisherFactory := func(identity identity.Identity) communication.DialogEstablisher {
		return &fakeDialog{}
	}

	tc.fakeOpenVpn = &fakeOpenvpnClient{
		false,
		make(chan int, 1),
		make(chan int, 1),
		nil,
		nil,
	}
	fakeVpnClientFactory := func(vpnSession session.SessionDto, identity identity.Identity, callback state.ClientStateCallback) (openvpn.Client, error) {
		tc.fakeOpenVpn.stateCallback = callback
		return tc.fakeOpenVpn, nil
	}
	tc.fakeStatsKeeper = &fakeSessionStatsKeeper{}

	tc.connManager = NewManager(tc.fakeDiscoveryClient, dialogEstablisherFactory, fakeVpnClientFactory, tc.fakeStatsKeeper)
}

func (tc *testContext) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", nil}, tc.connManager.Status())
}

func (tc *testContext) TestWithUnknownProviderConnectionIsNotMade() {
	noProposalsError := errors.New("provider has no service proposals")

	assert.Error(tc.T(), tc.connManager.Connect(myID, identity.FromAddress("unknown-node")))
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", noProposalsError}, tc.connManager.Status())

	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestOnConnectErrorStatusIsNotConnectedAndLastErrorIsSetAndSessionStartIsNotMarked() {
	fatalVpnError := errors.New("fatal connection error")
	tc.fakeOpenVpn.onConnectReturnError = fatalVpnError

	assert.Error(tc.T(), tc.connManager.Connect(myID, activeProviderID))
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", fatalVpnError}, tc.connManager.Status())

	assert.False(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(myID, activeProviderID)

	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), ConnectionStatus{Connected, "vpn-session-id", nil}, tc.connManager.Status())
}

func (tc *testContext) TestWhenManagerMadeConnectionSessionStartIsMarked() {
	err := tc.connManager.Connect(myID, activeProviderID)

	assert.NoError(tc.T(), err)

	assert.True(tc.T(), tc.fakeStatsKeeper.sessionStartMarked)
}

func (tc *testContext) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeOpenVpn.delayableAction()
	go func() {
		tc.connManager.Connect(myID, activeProviderID)
	}()
	tc.fakeOpenVpn.waitForDelayState()
	assert.Equal(tc.T(), ConnectionStatus{Connecting, "", nil}, tc.connManager.Status())
}

func (tc *testContext) TestStatusReportsDisconnectingThenNotConnected() {
	err := tc.connManager.Connect(myID, activeProviderID)
	tc.fakeOpenVpn.reportState(openvpn.STATE_CONNECTED)

	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), ConnectionStatus{Connected, "vpn-session-id", nil}, tc.connManager.Status())

	tc.fakeOpenVpn.delayableAction()
	disconnectCompleted := sync.WaitGroup{}
	disconnectCompleted.Add(1)
	go func() {
		tc.connManager.Disconnect()
		disconnectCompleted.Done()
	}()

	tc.fakeOpenVpn.waitForDelayState()
	assert.Equal(tc.T(), ConnectionStatus{Disconnecting, "", nil}, tc.connManager.Status())
	tc.fakeOpenVpn.resumeAction()
	tc.fakeOpenVpn.reportState(openvpn.STATE_EXITING)
	disconnectCompleted.Wait()
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", nil}, tc.connManager.Status())
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(testContext))
}

type fakeOpenvpnClient struct {
	delayAction               bool
	delayStateEnteredNotifier chan int
	resumeFromDelay           chan int
	onConnectReturnError      error
	stateCallback             state.ClientStateCallback
}

func (foc *fakeOpenvpnClient) Start() error {
	if foc.delayAction {
		foc.delayStateEnteredNotifier <- 1
		<-foc.resumeFromDelay
	}
	return foc.onConnectReturnError
}

func (foc *fakeOpenvpnClient) Wait() error {
	return nil
}

func (foc *fakeOpenvpnClient) Stop() error {
	if foc.delayAction {
		foc.delayStateEnteredNotifier <- 1
		<-foc.resumeFromDelay
	}
	return nil
}

func (foc *fakeOpenvpnClient) delayableAction() {
	foc.delayAction = true
}

func (foc *fakeOpenvpnClient) waitForDelayState() {
	<-foc.delayStateEnteredNotifier
}

func (foc *fakeOpenvpnClient) resumeAction() {
	foc.resumeFromDelay <- 1
}

func (foc *fakeOpenvpnClient) reportState(state openvpn.State) {
	foc.stateCallback(state)
}

type fakeDialog struct {
	peerId identity.Identity
}

func (fd *fakeDialog) CreateDialog(peerID identity.Identity, peerContact dto_discovery.Contact) (communication.Dialog, error) {
	fd.peerId = peerID
	return fd, nil
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
				ID:     "vpn-session-id",
				Config: "vpn-session-config",
			},
		},
		nil
}

type fakeSessionStatsKeeper struct {
	sessionStartMarked bool
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
