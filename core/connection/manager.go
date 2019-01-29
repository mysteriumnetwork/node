/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package connection

import (
	"context"
	"errors"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
)

const managerLogPrefix = "[connection-manager] "

var (
	// ErrNoConnection error indicates that action applied to manager expects active connection (i.e. disconnect)
	ErrNoConnection = errors.New("no connection exists")
	// ErrAlreadyExists error indicates that action applied to manager expects no active connection (i.e. connect)
	ErrAlreadyExists = errors.New("connection already exists")
	// ErrConnectionCancelled indicates that connection in progress was cancelled by request of api user
	ErrConnectionCancelled = errors.New("connection was cancelled")
	// ErrConnectionFailed indicates that Connect method didn't reach "Connected" phase due to connection error
	ErrConnectionFailed = errors.New("connection has failed")
	// ErrUnsupportedServiceType indicates that target proposal contains unsupported service type
	ErrUnsupportedServiceType = errors.New("unsupported service type in proposal")
)

// Creator creates new connection by given options and uses state channel to report state changes
type Creator func(serviceType string, stateChannnel StateChannel, statisticsChannel StatisticsChannel) (Connection, error)

// SessionInfo contains all the relevant info of the current session
type SessionInfo struct {
	SessionID  session.ID
	ConsumerID identity.Identity
	Proposal   market.ServiceProposal
}

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, args ...interface{})
}

type connectionManager struct {
	//these are passed on creation
	newDialog        DialogCreator
	newPromiseIssuer PromiseIssuerCreator
	newConnection    Creator
	eventPublisher   Publisher

	//these are populated by Connect at runtime
	ctx             context.Context
	mutex           sync.RWMutex
	status          ConnectionStatus
	sessionInfo     SessionInfo
	cleanConnection func()
}

// NewManager creates connection manager with given dependencies
func NewManager(
	dialogCreator DialogCreator,
	promiseIssuerCreator PromiseIssuerCreator,
	connectionCreator Creator,
	eventPublisher Publisher,
) *connectionManager {
	return &connectionManager{
		newDialog:        dialogCreator,
		newPromiseIssuer: promiseIssuerCreator,
		newConnection:    connectionCreator,
		status:           statusNotConnected(),
		cleanConnection:  warnOnClean,
		eventPublisher:   eventPublisher,
	}
}

func (manager *connectionManager) Connect(consumerID identity.Identity, proposal market.ServiceProposal, params ConnectParams) (err error) {
	if manager.status.State != NotConnected {
		return ErrAlreadyExists
	}

	manager.mutex.Lock()
	manager.ctx, manager.cleanConnection = context.WithCancel(context.Background())
	manager.status = statusConnecting()
	manager.mutex.Unlock()
	defer func() {
		if err != nil {
			manager.mutex.Lock()
			manager.status = statusNotConnected()
			manager.mutex.Unlock()
		}
	}()

	err = manager.startConnection(consumerID, proposal, params)
	if err == context.Canceled {
		return ErrConnectionCancelled
	}
	return err
}

func (manager *connectionManager) startConnection(consumerID identity.Identity, proposal market.ServiceProposal, params ConnectParams) (err error) {
	manager.mutex.Lock()
	cancelCtx := manager.cleanConnection
	manager.mutex.Unlock()

	var cancel []func()
	defer func() {
		manager.cleanConnection = func() {
			manager.status = statusDisconnecting()
			cancelCtx()
			for _, f := range cancel {
				f()
			}
		}
		if err != nil {
			log.Info(managerLogPrefix, "Cancelling connection initiation")
			defer manager.cleanConnection()
		}
	}()

	providerID := identity.FromAddress(proposal.ProviderID)
	dialog, err := manager.newDialog(consumerID, providerID, proposal.ProviderContacts[0])
	if err != nil {
		return err
	}
	cancel = append(cancel, func() { dialog.Close() })

	stateChannel := make(chan State, 10)
	statisticsChannel := make(chan consumer.SessionStatistics, 10)

	connection, err := manager.newConnection(proposal.ServiceType, stateChannel, statisticsChannel)
	if err != nil {
		return err
	}

	sessionCreateConfig, err := connection.GetConfig()
	if err != nil {
		return err
	}

	sessionID, sessionConfig, err := session.RequestSessionCreate(dialog, proposal.ID, sessionCreateConfig)
	if err != nil {
		return err
	}

	cancel = append(cancel, func() { session.RequestSessionDestroy(dialog, sessionID) })

	// set the session info for future use
	manager.sessionInfo = SessionInfo{
		SessionID:  sessionID,
		ConsumerID: consumerID,
		Proposal:   proposal,
	}

	manager.eventPublisher.Publish(SessionEventTopic, SessionEvent{
		Status:      SessionCreatedStatus,
		SessionInfo: manager.sessionInfo,
	})

	cancel = append(cancel, func() {
		manager.eventPublisher.Publish(SessionEventTopic, SessionEvent{
			Status:      SessionEndedStatus,
			SessionInfo: manager.sessionInfo,
		})
	})

	promiseIssuer := manager.newPromiseIssuer(consumerID, dialog)
	err = promiseIssuer.Start(proposal)
	if err != nil {
		return err
	}
	cancel = append(cancel, func() { promiseIssuer.Stop() })

	connectOptions := ConnectOptions{
		SessionID:     sessionID,
		SessionConfig: sessionConfig,
		ConsumerID:    consumerID,
		ProviderID:    providerID,
		Proposal:      proposal,
	}

	if err = connection.Start(connectOptions); err != nil {
		return err
	}
	cancel = append(cancel, connection.Stop)

	//consume statistics right after start - openvpn3 will publish them even before connected state
	go manager.consumeStats(statisticsChannel)
	err = manager.waitForConnectedState(stateChannel, sessionID)
	if err != nil {
		return err
	}

	if !params.DisableKillSwitch {
		// TODO: Implement fw based kill switch for respective OS
		// we may need to wait for tun device setup to be finished
		firewall.NewKillSwitch().Enable()
	}

	go manager.consumeConnectionStates(stateChannel)
	go connectionWaiter(connection, dialog, promiseIssuer)
	return nil
}

func (manager *connectionManager) Status() ConnectionStatus {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	return manager.status
}

func (manager *connectionManager) Disconnect() error {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	if manager.status.State == NotConnected {
		return ErrNoConnection
	}
	manager.cleanConnection()
	return nil
}

func warnOnClean() {
	log.Warn(managerLogPrefix, "Trying to close when there is nothing to close. Possible bug or race condition")
}

func connectionWaiter(connection Connection, dialog communication.Dialog, promiseIssuer PromiseIssuer) {
	err := connection.Wait()
	if err != nil {
		log.Warn(managerLogPrefix, "Connection exited with error: ", err)
	} else {
		log.Info(managerLogPrefix, "Connection exited")
	}

	promiseIssuer.Stop()
	dialog.Close()
}

func (manager *connectionManager) waitForConnectedState(stateChannel <-chan State, sessionID session.ID) error {
	for {
		select {
		case state, more := <-stateChannel:
			if !more {
				return ErrConnectionFailed
			}

			switch state {
			case Connected:
				manager.onStateChanged(state)
				return nil
			default:
				manager.onStateChanged(state)
			}
		case <-manager.ctx.Done():
			return manager.ctx.Err()
		}
	}
}

func (manager *connectionManager) consumeConnectionStates(stateChannel <-chan State) {
	for state := range stateChannel {
		manager.onStateChanged(state)
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.status = statusNotConnected()
	log.Debug(managerLogPrefix, "State updater stopCalled")
}

func (manager *connectionManager) consumeStats(statisticsChannel <-chan consumer.SessionStatistics) {
	for stats := range statisticsChannel {
		manager.eventPublisher.Publish(StatisticsEventTopic, stats)
	}
}

func (manager *connectionManager) onStateChanged(state State) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.eventPublisher.Publish(StateEventTopic, StateEvent{
		State:       state,
		SessionInfo: manager.sessionInfo,
	})

	switch state {
	case Connected:
		manager.status = statusConnected(manager.sessionInfo.SessionID)
	case Reconnecting:
		manager.status = statusReconnecting()
	}
}
