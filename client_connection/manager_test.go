package client_connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	mysterium_api_client "github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type test_context struct {
	suite.Suite
	connManager         *connectionManager
	fakeDiscoveryClient server.Client
	fakeOpenVpn         *fake_openvpn_client
}

func (tc *test_context) SetupTest() {

	tc.fakeDiscoveryClient = server.NewClientFake()

	serviceProposal := service_discovery.NewServiceProposal(identity.FromAddress("vpn-node-1"), dto.Contact{})
	tc.fakeDiscoveryClient.NodeRegister(serviceProposal)

	dialogEstablisherFactory := func(identity identity.Identity) communication.DialogEstablisher {
		return &fake_dialog{}
	}

	tc.fakeOpenVpn = &fake_openvpn_client{make(chan int, 1), nil}
	fakeVpnClientFactory := func(vpnSession *session.VpnSession, session *mysterium_api_client.Session) (openvpn.Client, error) {
		return tc.fakeOpenVpn, nil
	}

	tc.connManager = NewManager(tc.fakeDiscoveryClient, dialogEstablisherFactory, fakeVpnClientFactory)
}

func (tc *test_context) TestWhenNoConnectionIsMadeStatusIsNotConnected() {
	assert.Equal(tc.T(), ConnectionStatus{NOT_CONNECTED, "", nil}, tc.connManager.Status())
}

func (tc *test_context) TestOnConnectErrorStatusIsNotConnectedAndLastErrorIsSet() {
	fatalVpnError := errors.New("fatal connection error")
	tc.fakeOpenVpn.onConnectReturnError = fatalVpnError
	tc.fakeOpenVpn.resumeStart()

	assert.Error(tc.T(), tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1"))
	assert.Equal(tc.T(), ConnectionStatus{NOT_CONNECTED, "", fatalVpnError}, tc.connManager.Status())
}

func (tc *test_context) TestWhenManagerMadeConnectionStatusReturnsConnectedStateAndSessionId() {
	tc.fakeOpenVpn.resumeStart()

	err := tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1")

	assert.NoError(tc.T(), err)
	assert.Equal(tc.T(), ConnectionStatus{CONNECTED, "vpn-node-1-session", nil}, tc.connManager.Status())
}

func (tc *test_context) TestStatusReportsConnectingWhenConnectionIsInProgress() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		wg.Done()
		tc.connManager.Connect(identity.FromAddress("identity-1"), "vpn-node-1")
	}()
	//wait for go function actually start, to avoid race condition, when we query Status before Connect call even begins.
	wg.Wait()
	assert.Equal(tc.T(), ConnectionStatus{NEGOTIATING, "", nil}, tc.connManager.Status())
	tc.fakeOpenVpn.resumeStart()
}

func (tc *test_context) TearDownTest() {

}

func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(test_context))
}

type fake_openvpn_client struct {
	connectionDelay      chan int
	onConnectReturnError error
}

func (foc *fake_openvpn_client) resumeStart() {
	foc.connectionDelay <- 1
}

func (foc *fake_openvpn_client) Start() error {
	<-foc.connectionDelay
	return foc.onConnectReturnError
}

func (foc *fake_openvpn_client) Wait() error {
	return nil
}

func (foc *fake_openvpn_client) Stop() error {
	return nil
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
			true,
			"Everything is great!",
			session.VpnSession{
				"vpn-session-id",
				"vpn-session-config"},
		},
		nil
}
