package client_connection

import (
	"errors"
	"github.com/mysterium/node/bytescount_client"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"path/filepath"
	"time"
)

type DialogEstablisherFactory func(identity identity.Identity) communication.DialogEstablisher

type VpnClientFactory func(vpnSession session.VpnSession, session dto.Session) (openvpn.Client, error)

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

func (manager *connectionManager) Connect(identity identity.Identity, nodeKey string) error {
	manager.status = statusConnecting()

	proposals, err := manager.mysteriumClient.Proposals(nodeKey)
	if err != nil {
		manager.status = statusError(err)
		return err
	}
	if len(proposals) == 0 {
		err = errors.New("node has no service proposals")
		manager.status = statusError(err)
		return err
	}
	proposal := proposals[0]

	// TODO: pass in selected proposal to session creation
	clientSession, err := manager.mysteriumClient.SessionCreate(nodeKey)
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	dialogEstablisher := manager.dialogEstablisherFactory(identity)
	manager.dialog, err = dialogEstablisher.CreateDialog(proposal.ProviderContacts[0])
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	vpnSession, err := session.RequestSessionCreate(manager.dialog, proposal.Id)
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	manager.vpnClient, err = manager.vpnClientFactory(*vpnSession, clientSession)

	if err := manager.vpnClient.Start(); err != nil {
		manager.status = statusError(err)
		return err
	}
	manager.status = statusConnected(clientSession.Id)
	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	return manager.status
}

func (manager *connectionManager) Disconnect() error {
	manager.dialog.Close()
	return manager.vpnClient.Stop()
}

func (manager *connectionManager) Wait() error {
	return manager.vpnClient.Wait()
}

func statusError(err error) ConnectionStatus {
	return ConnectionStatus{NotConnected, "", err}
}

func statusConnecting() ConnectionStatus {
	return ConnectionStatus{Connecting, "", nil}
}

func statusConnected(sessionId string) ConnectionStatus {
	return ConnectionStatus{Connected, sessionId, nil}
}

func statusNotConnected() ConnectionStatus {
	return ConnectionStatus{NotConnected, "", nil}
}

func ConfigureVpnClientFactory(mysteriumApiClient server.Client, vpnClientRuntimeDirectory string) VpnClientFactory {
	return func(vpnSession session.VpnSession, session dto.Session) (openvpn.Client, error) {
		vpnConfig, err := openvpn.NewClientConfigFromString(
			vpnSession.Config,
			filepath.Join(vpnClientRuntimeDirectory, "client.ovpn"),
		)
		if err != nil {
			return nil, err
		}

		vpnMiddlewares := []openvpn.ManagementMiddleware{
			bytescount_client.NewMiddleware(mysteriumApiClient, session.Id, 1*time.Minute),
		}
		return openvpn.NewClient(
			vpnConfig,
			vpnClientRuntimeDirectory,
			vpnMiddlewares...,
		), nil

	}
}
