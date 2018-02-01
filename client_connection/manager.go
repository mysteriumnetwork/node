package client_connection

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
)

type DialogEstablisherFactory func(identity identity.Identity) communication.DialogEstablisher

type VpnClientFactory func(session.SessionDto, identity.Identity, state.ClientStateCallback) (openvpn.Client, error)

type connectionManager struct {
	//these are passed on creation
	mysteriumClient  server.Client
	newDialogCreator DialogEstablisherFactory
	newVpnClient     VpnClientFactory
	statsKeeper      bytescount.SessionStatsKeeper
	//these are populated by Connect at runtime
	conn *connection
}

var (
	NoConnection  = errors.New("no connection exists")
	AlreadyExists = errors.New("connection already exists")
)

func NewManager(mysteriumClient server.Client, dialogEstablisherFactory DialogEstablisherFactory,
	vpnClientFactory VpnClientFactory, statsKeeper bytescount.SessionStatsKeeper) *connectionManager {
	return &connectionManager{
		mysteriumClient:  mysteriumClient,
		newDialogCreator: dialogEstablisherFactory,
		newVpnClient:     vpnClientFactory,
		statsKeeper:      statsKeeper,
		conn:             nil,
	}
}

func (manager *connectionManager) Connect(consumerID, providerID identity.Identity) error {
	if manager.conn != nil {
		return AlreadyExists
	}

	proposal, err := manager.findProposalByNode(providerID)
	if err != nil {
		return err
	}

	dialogCreator := manager.newDialogCreator(consumerID)
	dialog, err := dialogCreator.CreateDialog(providerID, proposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	vpnSession, err := session.RequestSessionCreate(dialog, proposal.ID)
	if err != nil {
		return err
	}

	vpnClient, err := manager.newVpnClient(*vpnSession, consumerID, manager.onVpnStateChanged)

	if err := vpnClient.Start(); err != nil {
		dialog.Close()
		return err
	}
	manager.conn = newConnection(dialog, vpnClient, vpnSession.ID)
	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	if manager.conn == nil {
		return statusNotConnected()
	}
	return manager.conn.status
}

func (manager *connectionManager) Disconnect() error {
	if manager.conn == nil {
		return NoConnection
	}
	manager.conn.close()
	return nil
}

// TODO this can be extraced as depencency later when node selection criteria will be clear
func (manager *connectionManager) findProposalByNode(nodeID identity.Identity) (*dto.ServiceProposal, error) {
	proposals, err := manager.mysteriumClient.FindProposals(nodeID.Address)
	if err != nil {
		return nil, err
	}
	if len(proposals) == 0 {
		err = errors.New("node has no service proposals")
		return nil, err
	}
	return &proposals[0], nil
}
