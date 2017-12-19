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
		ConnectionStatus{NOT_CONNECTED, ""},
	}
}

func (manager *connectionManager) Connect(identity identity.Identity, NodeKey string) error {
	manager.status = ConnectionStatus{NOT_CONNECTED, ""}
	session, err := manager.mysteriumClient.SessionCreate(NodeKey)
	if err != nil {
		return err
	}
	manager.status = ConnectionStatus{NEGOTIATING, session.Id}

	proposal := session.ServiceProposal

	dialogEstablisher := manager.dialogEstablisherFactory(identity)
	manager.dialog, err = dialogEstablisher.CreateDialog(proposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	vpnSession, err := vpn_session.RequestSessionCreate(manager.dialog, proposal.Id)
	if err != nil {
		return err
	}

	manager.vpnClient, err = manager.vpnClientFactory(vpnSession, &session)

	if err := manager.vpnClient.Start(); err != nil {
		return err
	}
	manager.status = ConnectionStatus{CONNECTED, session.Id}
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
