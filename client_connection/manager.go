package client_connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
)

type DialogEstablisherFactory func(identity identity.Identity) communication.DialogEstablisher

type VpnClientFactory func(vpnSession *vpn_session.VpnSession, session *dto.Session) (openvpn.Client, error)

type connectionManager struct {
	//these are passed on creation
	mysteriumClient          server.Client
	dialogEstablisherFactory DialogEstablisherFactory
	vpnClientFactory         VpnClientFactory
	//these are populated by Connect at runtime
	dialog    communication.Dialog
	vpnClient openvpn.Client
	status    ConnectionStatus
}

func NewManager(mysteriumClient server.Client, dialogEstablisherFactory DialogEstablisherFactory, vpnClientFactory VpnClientFactory) *connectionManager {
	return &connectionManager{
		mysteriumClient,
		dialogEstablisherFactory,
		vpnClientFactory,
		nil,
		nil,
		statusNotConnected(),
	}
}

func (manager *connectionManager) Connect(identity identity.Identity, NodeKey string) error {
	manager.status = statusConnecting()
	session, err := manager.mysteriumClient.SessionCreate(NodeKey)
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	proposal := session.ServiceProposal

	dialogEstablisher := manager.dialogEstablisherFactory(identity)
	manager.dialog, err = dialogEstablisher.CreateDialog(proposal.ProviderContacts[0])
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	vpnSession, err := vpn_session.RequestSessionCreate(manager.dialog, proposal.Id)
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	manager.vpnClient, err = manager.vpnClientFactory(vpnSession, &session)

	if err := manager.vpnClient.Start(); err != nil {
		manager.status = statusError(err)
		return err
	}
	manager.status = statusConnected(session.Id)
	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	return manager.status
}

func (manager *connectionManager) Disconnect() error {
	manager.dialog.Close()
	manager.vpnClient.Stop()
	return nil
}

func (manager *connectionManager) Wait() error {
	return manager.vpnClient.Wait()
}

func statusError(err error) ConnectionStatus {
	return ConnectionStatus{NOT_CONNECTED, "", err}
}

func statusConnecting() ConnectionStatus {
	return ConnectionStatus{NEGOTIATING, "", nil}
}

func statusConnected(sessionId string) ConnectionStatus {
	return ConnectionStatus{CONNECTED, sessionId, nil}
}

func statusNotConnected() ConnectionStatus {
	return ConnectionStatus{NOT_CONNECTED, "", nil}
}
