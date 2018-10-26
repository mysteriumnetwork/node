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
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
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

type connectionManager struct {
	//these are passed on creation
	mysteriumClient    server.Client
	newDialog          DialogCreator
	newPromiseIssuer   PromiseIssuerCreator
	connectionCreator  ConnectionCreator
	statsKeeper        stats.SessionStatsKeeper
	sessionsRepository SessionsRepository
	//these are populated by Connect at runtime
	ctx             context.Context
	mutex           sync.RWMutex
	status          ConnectionStatus
	cleanConnection func()
}

// NewManager creates connection manager with given dependencies
func NewManager(
	mysteriumClient server.Client,
	dialogCreator DialogCreator,
	promiseIssuerCreator PromiseIssuerCreator,
	connectionCreator ConnectionCreator,
	statsKeeper stats.SessionStatsKeeper,
	sessionsRepository SessionsRepository,
) *connectionManager {
	return &connectionManager{
		statsKeeper:        statsKeeper,
		mysteriumClient:    mysteriumClient,
		newDialog:          dialogCreator,
		newPromiseIssuer:   promiseIssuerCreator,
		connectionCreator:  connectionCreator,
		status:             statusNotConnected(),
		cleanConnection:    warnOnClean,
		sessionsRepository: sessionsRepository,
	}
}

func (manager *connectionManager) Connect(consumerID, providerID identity.Identity, params ConnectParams) (err error) {
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

	err = manager.startConnection(consumerID, providerID, params)
	if err == context.Canceled {
		return ErrConnectionCancelled
	}
	return err
}

func (manager *connectionManager) startConnection(consumerID, providerID identity.Identity, params ConnectParams) (err error) {
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

	proposal, err := manager.findProposalByProviderID(providerID)
	if err != nil {
		return err
	}

	dialog, err := manager.newDialog(consumerID, providerID, proposal.ProviderContacts[0])
	if err != nil {
		return err
	}
	cancel = append(cancel, func() { dialog.Close() })

	sessionID, sessionConfig, err := session.RequestSessionCreate(dialog, proposal.ID)
	if err != nil {
		return err
	}

	promiseIssuer := manager.newPromiseIssuer(consumerID, dialog)
	err = promiseIssuer.Start(proposal)
	if err != nil {
		return err
	}
	cancel = append(cancel, func() { promiseIssuer.Stop() })

	stateChannel := make(chan State, 10)

	connectOptions := ConnectOptions{
		SessionID:     sessionID,
		SessionConfig: sessionConfig,
		ConsumerID:    consumerID,
		ProviderID:    providerID,
		Proposal:      proposal,
	}

	connection, err := manager.connectionCreator.CreateConnection(connectOptions, stateChannel)
	if err != nil {
		return err
	}

	err = manager.saveSession(connectOptions)
	if err != nil {
		return err
	}

	if err = connection.Start(); err != nil {
		return err
	}
	cancel = append(cancel, connection.Stop)

	err = manager.waitForConnectedState(stateChannel, sessionID)
	if err != nil {
		return err
	}

	if !params.DisableKillSwitch {
		// TODO: Implement fw based kill switch for respective OS
		// we may need to wait for tun device to bet setup
		firewall.NewKillSwitch().Enable()
	}

	go connectionWaiter(connection, dialog, promiseIssuer)
	go manager.consumeConnectionStates(stateChannel, sessionID)
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

// TODO this can be extracted as dependency later when node selection criteria will be clear
func (manager *connectionManager) findProposalByProviderID(providerID identity.Identity) (proposal dto.ServiceProposal, err error) {
	proposals, err := manager.mysteriumClient.FindProposals(providerID.Address)
	if err != nil {
		return
	}
	if len(proposals) == 0 {
		err = errors.New("provider has no service proposals")
		return
	}

	proposal = proposals[0]
	return
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
				manager.onStateChanged(state, sessionID)
				return nil
			default:
				manager.onStateChanged(state, sessionID)
			}
		case <-manager.ctx.Done():
			return manager.ctx.Err()
		}
	}
}

func (manager *connectionManager) consumeConnectionStates(stateChannel <-chan State, sessionID session.ID) {
	for state := range stateChannel {
		manager.onStateChanged(state, sessionID)
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.status = statusNotConnected()
	log.Debug(managerLogPrefix, "State updater stopCalled")
}

func (manager *connectionManager) onStateChanged(state State, sessionID session.ID) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	switch state {
	case Connected:
		manager.statsKeeper.MarkSessionStart()
		manager.status = statusConnected(sessionID)
	case Disconnecting:
		manager.statsKeeper.MarkSessionEnd()
	case Reconnecting:
		manager.status = statusReconnecting()
	}
}

func (manager *connectionManager) saveSession(connectOptions ConnectOptions) error {
	providerCountry := connectOptions.Proposal.ServiceDefinition.GetLocation().Country
	se := Session{
		SessionID:       connectOptions.SessionID,
		ProviderID:      connectOptions.ProviderID,
		ServiceType:     connectOptions.Proposal.ServiceType,
		ProviderCountry: providerCountry,
		TimeStarted:     time.Now(),
	}

	err := manager.sessionsRepository.Save(se)
	if err != nil {
		return err
	}
	return nil
}
