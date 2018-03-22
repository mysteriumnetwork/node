package connection

import (
	"errors"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"sync"
)

const managerLogPrefix = "[Connection manager] "

var (
	// ErrNoConnection error indicates that action applied to manager expects active connection (i.e. disconnect)
	ErrNoConnection = errors.New("no connection exists")
	// ErrAlreadyExists error indicates that aciton applieto to manager expects no active connection (i.e. connect)
	ErrAlreadyExists = errors.New("connection already exists")
)

type connectionManager struct {
	//these are passed on creation
	mysteriumClient server.Client
	newDialog       DialogCreator
	newVpnClient    VpnClientCreator
	statsKeeper     bytescount.SessionStatsKeeper
	//these are populated by Connect at runtime
	status      ConnectionStatus
	closeAction func()
}

func warnOnClose() {
	log.Warn(managerLogPrefix, "WARNING! Trying to close when there is nothing to close. Possible bug or race condition")
}

// NewManager creates connection manager with given dependencies
func NewManager(mysteriumClient server.Client, dialogCreator DialogCreator,
	vpnClientCreator VpnClientCreator, statsKeeper bytescount.SessionStatsKeeper) *connectionManager {
	return &connectionManager{
		mysteriumClient: mysteriumClient,
		newDialog:       dialogCreator,
		newVpnClient:    vpnClientCreator,
		statsKeeper:     statsKeeper,
		status:          statusNotConnected(),
		closeAction:     warnOnClose,
	}
}

func (manager *connectionManager) Connect(consumerID, providerID identity.Identity) (err error) {
	if manager.status.State != NotConnected {
		return ErrAlreadyExists
	}

	manager.status = statusConnecting()
	defer func() {
		if err != nil {
			manager.closeAction()
			manager.status = statusNotConnected()
		}
	}()

	closeRequest := make(chan int)
	closeChannelOnce := sync.Once{}
	manager.closeAction = func() {
		closeChannelOnce.Do(func() {
			log.Info(managerLogPrefix, "Closing active connection")
			manager.status = statusDisconnecting()
			close(closeRequest)
		})
	}
	proposal, err := manager.findProposalByProviderID(providerID)
	if err != nil {
		return err
	}

	dialog, err := manager.newDialog(consumerID, providerID, proposal.ProviderContacts[0])
	if err != nil {
		return err
	}
	go dialogCloser(closeRequest, dialog)

	vpnSession, err := session.RequestSessionCreate(dialog, proposal.ID)
	if err != nil {
		return err
	}

	openvpnClient, err := manager.newVpnClient(
		*vpnSession,
		consumerID,
		providerID,
		func(state openvpn.State) {
			switch state {
			case openvpn.ConnectedState:
				manager.statsKeeper.MarkSessionStart()
				manager.status = statusConnected(vpnSession.ID)
			case openvpn.ExitingState:
				manager.statsKeeper.MarkSessionEnd()
			case openvpn.ReconnectingState:
				manager.status = statusReconnecting()
			}
		},
	)
	if err != nil {
		return err
	}

	if err = openvpnClient.Start(); err != nil {
		return err
	}

	go clientStopper(closeRequest, openvpnClient)
	go manager.clientWaiter(openvpnClient)

	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	return manager.status
}

func (manager *connectionManager) Disconnect() error {
	if manager.status.State == NotConnected {
		return ErrNoConnection
	}
	manager.closeAction()
	return nil
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

func dialogCloser(closeRequest <-chan int, dialog communication.Dialog) {
	<-closeRequest
	log.Info(managerLogPrefix, "Closing dialog")
	dialog.Close()
}

func clientStopper(closeRequest <-chan int, openvpnClient openvpn.Client) {
	<-closeRequest
	log.Info(managerLogPrefix, "Stopping openvpn client")
	err := openvpnClient.Stop()
	if err != nil {
		log.Error(managerLogPrefix, "Failed to stop openvpn client: ", err)
	}
}

func (manager *connectionManager) clientWaiter(openvpnClient openvpn.Client) {
	err := openvpnClient.Wait()
	if err != nil {
		log.Error(managerLogPrefix, "Openvpn client exited with error: ", err)
	} else {
		log.Info(managerLogPrefix, "Openvpn client exited")
	}
	manager.status = statusNotConnected()
}
