package client_connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/session"
	"path/filepath"
	"time"
)

type DialogEstablisherFactory func(identity identity.Identity) communication.DialogEstablisher

type VpnClientFactory func(vpnSession session.SessionDto, identity identity.Identity) (openvpn.Client, error)

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

func (manager *connectionManager) Connect(id identity.Identity, nodeKey string) error {
	manager.status = statusConnecting()

	proposals, err := manager.mysteriumClient.FindProposals(nodeKey)
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

	dialogEstablisher := manager.dialogEstablisherFactory(id)
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

	manager.vpnClient, err = manager.vpnClientFactory(*vpnSession, id)
	if err := manager.vpnClient.Start(); err != nil {
		manager.status = statusError(err)
		return err
	}

	manager.status = statusConnected(vpnSession.Id)
	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	return manager.status
}

func (manager *connectionManager) Disconnect() error {
	manager.status = statusDisconnecting()

	if manager.vpnClient != nil {
		if err := manager.vpnClient.Stop(); err != nil {
			return err
		}
	}
	if manager.dialog != nil {
		if err := manager.dialog.Close(); err != nil {
			return err
		}
	}

	manager.status = statusNotConnected()
	return nil
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

func statusConnected(sessionId session.SessionId) ConnectionStatus {
	return ConnectionStatus{Connected, sessionId, nil}
}

func statusNotConnected() ConnectionStatus {
	return ConnectionStatus{NotConnected, "", nil}
}

func statusDisconnecting() ConnectionStatus {
	return ConnectionStatus{Disconnecting, "", nil}
}

func ConfigureVpnClientFactory(mysteriumApiClient server.Client, vpnClientRuntimeDirectory string, signerFactory identity.SignerFactory) VpnClientFactory {
	return func(vpnSession session.SessionDto, id identity.Identity) (openvpn.Client, error) {
		vpnConfig, err := openvpn.NewClientConfigFromString(
			vpnSession.Config,
			filepath.Join(vpnClientRuntimeDirectory, "client.ovpn"),
		)
		if err != nil {
			return nil, err
		}

		statsSender := bytescount.NewSessionStatsSender(mysteriumApiClient, vpnSession.Id, signerFactory(id))
		authenticator := auth.NewAuthenticator()
		vpnMiddlewares := []openvpn.ManagementMiddleware{
			bytescount.NewMiddleware(statsSender, 1*time.Minute),
			auth.NewMiddleware(authenticator),
		}
		return openvpn.NewClient(
			vpnConfig,
			vpnClientRuntimeDirectory,
			vpnMiddlewares...,
		), nil

	}
}
