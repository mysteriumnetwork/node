package client_connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type test_context struct {
	suite.Suite
	connManager         *connectionManager
	fakeDiscoveryClient *server.ClientFake
	fakeOpenVpn         *fake_openvpn_client
}

func (tc *test_context) SetupTest() {

	tc.fakeDiscoveryClient = server.NewClientFake()

	serviceProposal := service_discovery.NewServiceProposal(identity.FromAddress("vpn-node-1"), dto.Contact{})
	tc.fakeDiscoveryClient.NodeRegister(serviceProposal)

	dialogEstablisherFactory := func(identity identity.Identity) communication.DialogEstablisher {
		return &fake_dialog{}
	}

	tc.fakeOpenVpn = &fake_openvpn_client{
		false,
		make(chan int, 1),
		make(chan int, 1),
		nil,
	}
	var fakeVpnClientFactory VpnClientFactory = func(vpnSession session.SessionDto) (openvpn.Client, error) {
		return tc.fakeOpenVpn, nil
	}

	tc.connManager = NewManager(tc.fakeDiscoveryClient, dialogEstablisherFactory, fakeVpnClientFactory)
}

func (tc *test_context) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", nil}, tc.connManager.Status())
}

func (tc *test_context) TestWithUnknownNodeKey() {
	noProposalsError := errors.New("node has no service proposals")

	assert.Error(tc.T(), tc.connManager.Connect(identity.FromAddress("identity-1"), "unknown-node"))
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", noProposalsError}, tc.connManager.Status())
}

func (tc *test_context) TestOnConnectErrorStatusIsNotConnectedAndLastErrorIsSet() {
	fatalVpnError := errors.New("fatal connection error")
	tc.fakeOpenVpn.onConnectReturnError = fatalVpnError

	assert.Error(tc.T(), tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1"))
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", fatalVpnError}, tc.connManager.Status())
}

func (tc *test_context) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	err := tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1")

	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), ConnectionStatus{Connected, "vpn-session-id", nil}, tc.connManager.Status())
}

func (tc *test_context) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	tc.fakeOpenVpn.delayableAction()
	go func() {
		tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1")
	}()
	tc.fakeOpenVpn.waitForDelayState()
	assert.Equal(tc.T(), ConnectionStatus{Connecting, "", nil}, tc.connManager.Status())
}

func (tc *test_context) TestStatusReportsDisconnectingThenNotConnected() {
	err := tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1")

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
	disconnectCompleted.Wait()
	assert.Equal(tc.T(), ConnectionStatus{NotConnected, "", nil}, tc.connManager.Status())
}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(test_context))
}

type fake_openvpn_client struct {
	delayAction               bool
	delayStateEnteredNotifier chan int
	resumeFromDelay           chan int
	onConnectReturnError      error
}

func (foc *fake_openvpn_client) Start() error {
	if foc.delayAction {
		foc.delayStateEnteredNotifier <- 1
		<-foc.resumeFromDelay
	}
	return foc.onConnectReturnError
}

func (foc *fake_openvpn_client) Wait() error {
	return nil
}

func (foc *fake_openvpn_client) Stop() error {
	if foc.delayAction {
		foc.delayStateEnteredNotifier <- 1
		<-foc.resumeFromDelay
	}
	return nil
}

func (foc *fake_openvpn_client) delayableAction() {
	foc.delayAction = true
}

func (foc *fake_openvpn_client) waitForDelayState() {
	<-foc.delayStateEnteredNotifier
}

func (foc *fake_openvpn_client) resumeAction() {
	foc.resumeFromDelay <- 1
}

type fake_dialog struct {
}

func (fd *fake_dialog) CreateDialog(contact dto.Contact) (communication.Dialog, error) {
	return fd, nil
}

func (fd *fake_dialog) Close() error {
	return nil
}

func (fd *fake_dialog) Receive(consumer communication.MessageConsumer) error {
	return nil
}
func (fd *fake_dialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}

func (fd *fake_dialog) Send(producer communication.MessageProducer) error {
	return nil
}

func (fd *fake_dialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return &session.SessionCreateResponse{
			Success: true,
			Message: "Everything is great!",
			Session: session.SessionDto{
				Id:     "vpn-session-id",
				Config: "vpn-session-config",
			},
		},
		nil
}
