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
	// ErrConnectionCancelled indicates that connection in progress was cancelled by request of api user
	ErrConnectionCancelled = errors.New("connection was cancelled")
	// ErrOpenvpnProcessDied indicates that Connect method didn't reach "Connected" phase due to openvpn error
	ErrOpenvpnProcessDied = errors.New("openvpn process died")
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

	stateChannel := make(chan openvpn.State, 10)
	openvpnClient, err := manager.newVpnClient(
		*vpnSession,
		consumerID,
		providerID,
		channelToStateCallbackAdapter(stateChannel),
	)
	if err != nil {
		return err
	}

	if err = openvpnClient.Start(); err != nil {
		return err
	}

	go manager.clientWaiter(openvpnClient, stateChannel)

connectBlocker:
	for {
		select {
		case state := <-stateChannel:
			switch state {
			case openvpn.ConnectedState:
				manager.onStateChanged(state, vpnSession.ID)
				break connectBlocker
			case "":
				//empty "state" means that channel was closed by clientWaiter (openvpn process exited)
				return ErrOpenvpnProcessDied
			default:
				manager.onStateChanged(state, vpnSession.ID)
			}
		case <-closeRequest:
			return ErrConnectionCancelled
		}
	}
	go func() {
		for state := range stateChannel {
			manager.onStateChanged(state, vpnSession.ID)
		}
		manager.status = statusNotConnected()
		log.Info("State updater stopped")
	}()

	go clientStopper(openvpnClient, closeRequest)
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

func clientStopper(openvpnClient openvpn.Client, closeRequest <-chan int) {
	<-closeRequest
	log.Info(managerLogPrefix, "Stopping openvpn client")
	err := openvpnClient.Stop()
	if err != nil {
		log.Error(managerLogPrefix, "Failed to stop openvpn client: ", err)
	}
}

func (manager *connectionManager) clientWaiter(openvpnClient openvpn.Client, stateChannel chan openvpn.State) {
	err := openvpnClient.Wait()
	if err != nil {
		log.Error(managerLogPrefix, "Openvpn client exited with error: ", err)
	} else {
		log.Info(managerLogPrefix, "Openvpn client exited")
	}
	close(stateChannel)
}
func (manager *connectionManager) onStateChanged(state openvpn.State, sessionID session.SessionID) {
	switch state {
	case openvpn.ConnectedState:
		manager.statsKeeper.MarkSessionStart()
		manager.status = statusConnected(sessionID)
	case openvpn.ExitingState:
		manager.statsKeeper.MarkSessionEnd()
	case openvpn.ReconnectingState:
		manager.status = statusReconnecting()
	}
}
