package connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
)

var (
	// ErrNoConnection error indicates that action applied to manager expects active connection (i.e. disconnect)
	ErrNoConnection = errors.New("no connection exists")
	// ErrAlreadyExists error indicates that aciton applieto to manager expects no active connection (i.e. connect)
	ErrAlreadyExists = errors.New("connection already exists")
)

type connectionManager struct {
	//these are passed on creation
	mysteriumClient      server.Client
	newDialogEstablisher DialogEstablisherCreator
	newVpnClient         VpnClientCreator
	statsKeeper          bytescount.SessionStatsKeeper
	//these are populated by Connect at runtime
	status        ConnectionStatus
	dialog        communication.Dialog
	openvpnClient openvpn.Client
	sessionID     session.SessionID
}

// NewManager creates connection manager with given dependencies
func NewManager(mysteriumClient server.Client, dialogEstablisherCreator DialogEstablisherCreator,
	vpnClientCreator VpnClientCreator, statsKeeper bytescount.SessionStatsKeeper) *connectionManager {
	return &connectionManager{
		mysteriumClient:      mysteriumClient,
		newDialogEstablisher: dialogEstablisherCreator,
		newVpnClient:         vpnClientCreator,
		statsKeeper:          statsKeeper,
		status:               statusNotConnected(),
	}
}

func (manager *connectionManager) Connect(consumerID, providerID identity.Identity) error {
	if manager.status.State != NotConnected {
		return ErrAlreadyExists
	}

	proposal, err := manager.findProposalByProviderID(providerID)
	if err != nil {
		return err
	}

	dialogEstablisher := manager.newDialogEstablisher(consumerID)
	manager.dialog, err = dialogEstablisher.EstablishDialog(providerID, proposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	vpnSession, err := session.RequestSessionCreate(manager.dialog, proposal.ID)
	if err != nil {
		return err
	}
	manager.sessionID = vpnSession.ID

	manager.openvpnClient, err = manager.newVpnClient(*vpnSession, consumerID, manager.onVpnStatusUpdate)
	if err != nil {
		manager.dialog.Close()
		return err
	}

	if err := manager.openvpnClient.Start(); err != nil {
		manager.dialog.Close()
		return err
	}
	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	return manager.status
}

func (manager *connectionManager) Disconnect() error {
	if manager.status.State == NotConnected {
		return ErrNoConnection
	}
	manager.status = statusDisconnecting()
	manager.openvpnClient.Stop()
	return nil
}

func (manager *connectionManager) onVpnStatusUpdate(vpnState openvpn.State) {
	switch vpnState {
	case openvpn.ConnectingState:
		manager.status = statusConnecting()
	case openvpn.ConnectedState:
		manager.statsKeeper.MarkSessionStart()
		manager.status = statusConnected(manager.sessionID)
	case openvpn.ExitingState:
		manager.status = statusNotConnected()
	case openvpn.ReconnectingState:
		manager.status = statusReconnecting()
	}
}

// TODO this can be extraced as depencency later when node selection criteria will be clear
func (manager *connectionManager) findProposalByProviderID(providerID identity.Identity) (*dto.ServiceProposal, error) {
	proposals, err := manager.mysteriumClient.FindProposals(providerID.Address)
	if err != nil {
		return nil, err
	}
	if len(proposals) == 0 {
		err = errors.New("provider has no service proposals")
		return nil, err
	}
	return &proposals[0], nil
}
