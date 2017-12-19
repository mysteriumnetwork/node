package client_connection

import (
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"path/filepath"
	"time"
)

type DialogEstablisherFactory func(identity identity.Identity) communication.DialogEstablisher

type connectionManager struct {
	//these are passed on creation
	mysteriumClient            server.Client
	dialogEstablisherFactory   DialogEstablisherFactory
	runtimeDirectory           string
	vpnClientConfigurationPath string
	//these are populated by Connect at runtime
	dialog    communication.Dialog
	vpnClient *openvpn.Client
}

func NewManager(mysteriumClient server.Client, dialogEstablisherFactory DialogEstablisherFactory, runtimeDirectory string) *connectionManager {
	vpnClientConfigFile := filepath.Join(runtimeDirectory, "client.ovpn")
	return &connectionManager{
		mysteriumClient,
		dialogEstablisherFactory,
		runtimeDirectory,
		vpnClientConfigFile,
		nil,
		nil,
	}
}

func (manager *connectionManager) Connect(identity identity.Identity, NodeKey string) error {

	session, err := manager.mysteriumClient.SessionCreate(NodeKey)
	if err != nil {
		return err
	}
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

	vpnConfig, err := openvpn.NewClientConfigFromString(
		vpnSession.Config,
		manager.vpnClientConfigurationPath,
	)
	if err != nil {
		return err
	}

	vpnMiddlewares := []openvpn.ManagementMiddleware{
		bytescount_client.NewMiddleware(manager.mysteriumClient, session.Id, 1*time.Minute),
	}
	manager.vpnClient = openvpn.NewClient(
		vpnConfig,
		manager.runtimeDirectory,
		vpnMiddlewares...,
	)
	if err := manager.vpnClient.Start(); err != nil {
		return err
	}

	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	return ConnectionStatus{}
}

func (manager *connectionManager) Disconnect() error {
	manager.dialog.Close()
	manager.vpnClient.Stop()
	return nil
}

func (manager *connectionManager) Wait() error {
	return manager.vpnClient.Wait()
}
