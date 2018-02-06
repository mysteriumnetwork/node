package client_connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	openvpnSession "github.com/mysterium/node/openvpn/session"
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
	statsKeeper              bytescount.SessionStatsKeeper
	//these are populated by Connect at runtime
	dialog    communication.Dialog
	vpnClient openvpn.Client
	status    ConnectionStatus
}

func NewManager(mysteriumClient server.Client, dialogEstablisherFactory DialogEstablisherFactory,
	vpnClientFactory VpnClientFactory, statsKeeper bytescount.SessionStatsKeeper) *connectionManager {
	return &connectionManager{
		mysteriumClient:          mysteriumClient,
		dialogEstablisherFactory: dialogEstablisherFactory,
		vpnClientFactory:         vpnClientFactory,
		statsKeeper:              statsKeeper,
		dialog:                   nil,
		vpnClient:                nil,
		status:                   statusNotConnected(),
	}
}

func (manager *connectionManager) Connect(consumerID identity.Identity, providerID identity.Identity) error {
	manager.status = statusConnecting()

	proposals, err := manager.mysteriumClient.FindProposals(providerID.Address)
	if err != nil {
		manager.status = statusError(err)
		return err
	}
	if len(proposals) == 0 {
		err = errors.New("provider has no service proposals")
		manager.status = statusError(err)
		return err
	}
	proposal := proposals[0]

	dialogEstablisher := manager.dialogEstablisherFactory(consumerID)
	manager.dialog, err = dialogEstablisher.CreateDialog(providerID, proposal.ProviderContacts[0])
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	vpnSession, err := session.RequestSessionCreate(manager.dialog, proposal.ID)
	if err != nil {
		manager.status = statusError(err)
		return err
	}

	manager.vpnClient, err = manager.vpnClientFactory(*vpnSession, consumerID)

	if err := manager.vpnClient.Start(); err != nil {
		manager.status = statusError(err)
		return err
	}

	manager.statsKeeper.MarkSessionStart()
	manager.status = statusConnected(vpnSession.ID)
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

func statusConnected(sessionID session.SessionID) ConnectionStatus {
	return ConnectionStatus{Connected, sessionID, nil}
}

func statusNotConnected() ConnectionStatus {
	return ConnectionStatus{NotConnected, "", nil}
}

func statusDisconnecting() ConnectionStatus {
	return ConnectionStatus{Disconnecting, "", nil}
}

func ConfigureVpnClientFactory(
	mysteriumAPIClient server.Client,
	configDirectory string,
	runtimeDirectory string,
	signerFactory identity.SignerFactory,
	statsKeeper bytescount.SessionStatsKeeper,
) VpnClientFactory {
	return func(vpnSession session.SessionDto, consumerID identity.Identity) (openvpn.Client, error) {
		vpnClientConfig, err := openvpn.NewClientConfigFromString(
			vpnSession.Config,
			filepath.Join(runtimeDirectory, "client.ovpn"),
			filepath.Join(configDirectory, "update-resolv-conf"),
			filepath.Join(configDirectory, "update-resolv-conf"),
		)
		if err != nil {
			return nil, err
		}

		signer := signerFactory(consumerID)

		statsSaver := bytescount.NewSessionStatsSaver(statsKeeper)
		statsSender := bytescount.NewSessionStatsSender(mysteriumAPIClient, vpnSession.ID, signer)
		statsHandler := bytescount.NewCompositeStatsHandler(statsSaver, statsSender)

		credentialsProvider := openvpnSession.SignatureCredentialsProvider(vpnSession.ID, signer)

		return openvpn.NewClient(
			vpnClientConfig,
			runtimeDirectory,
			bytescount.NewMiddleware(statsHandler, 1*time.Minute),
			auth.NewMiddleware(credentialsProvider),
		), nil
	}
}
