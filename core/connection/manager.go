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
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
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
	// ErrOpenvpnProcessDied indicates that Connect method didn't reach "Connected" phase due to openvpn error
	ErrOpenvpnProcessDied = errors.New("openvpn process died")
)

type connectionManager struct {
	//these are passed on creation
	mysteriumClient server.Client
	newDialog       DialogCreator
	newVpnClient    VpnClientCreator
	statsKeeper     stats.SessionStatsKeeper
	//these are populated by Connect at runtime
	ctx             context.Context
	mutex           sync.RWMutex
	status          ConnectionStatus
	cleanConnection func()
}

// NewManager creates connection manager with given dependencies
func NewManager(mysteriumClient server.Client, dialogCreator DialogCreator,
	vpnClientCreator VpnClientCreator, statsKeeper stats.SessionStatsKeeper) *connectionManager {
	return &connectionManager{
		mysteriumClient: mysteriumClient,
		newDialog:       dialogCreator,
		newVpnClient:    vpnClientCreator,
		statsKeeper:     statsKeeper,
		status:          statusNotConnected(),
		cleanConnection: warnOnClean,
	}
}

func (manager *connectionManager) Connect(consumerID, providerID identity.Identity, options ConnectOptions) (err error) {
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

	err = manager.startConnection(consumerID, providerID, options)
	if err == context.Canceled {
		return ErrConnectionCancelled
	}
	return err
}

func (manager *connectionManager) startConnection(consumerID, providerID identity.Identity, options ConnectOptions) (err error) {
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

	vpnSession, err := session.RequestSessionCreate(dialog, proposal.ID)
	if err != nil {
		return err
	}

	stateChannel := make(chan openvpn.State, 10)
	openvpnClient, err := manager.startOpenvpnClient(*vpnSession, consumerID, providerID, stateChannel, options)
	if err != nil {
		return err
	}
	cancel = append(cancel, openvpnClient.Stop)

	err = manager.waitForConnectedState(stateChannel, vpnSession.ID)
	if err != nil {
		return err
	}

	if !options.DisableKillSwitch {
		// TODO: Implement fw based kill switch for respective OS
		// we may need to wait for tun device to bet setup
		firewall.NewKillSwitch().Enable()
	}

	go openvpnClientWaiter(openvpnClient, dialog)
	go manager.consumeOpenvpnStates(stateChannel, vpnSession.ID)
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

func openvpnClientWaiter(openvpnClient openvpn.Process, dialog communication.Dialog) {
	err := openvpnClient.Wait()
	if err != nil {
		log.Warn(managerLogPrefix, "Openvpn client exited with error: ", err)
	} else {
		log.Info(managerLogPrefix, "Openvpn client exited")
	}
	dialog.Close()
}

func (manager *connectionManager) startOpenvpnClient(vpnSession session.SessionDto, consumerID, providerID identity.Identity, stateChannel chan openvpn.State, options ConnectOptions) (openvpn.Process, error) {
	openvpnClient, err := manager.newVpnClient(
		vpnSession,
		consumerID,
		providerID,
		channelToStateCallbackAdapter(stateChannel),
		options,
	)
	if err != nil {
		return nil, err
	}

	if err = openvpnClient.Start(); err != nil {
		return nil, err
	}

	return openvpnClient, nil
}

func (manager *connectionManager) waitForConnectedState(stateChannel <-chan openvpn.State, sessionID session.SessionID) error {
	for {
		select {
		case state, more := <-stateChannel:
			if !more {
				return ErrOpenvpnProcessDied
			}

			switch state {
			case openvpn.ConnectedState:
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

func (manager *connectionManager) consumeOpenvpnStates(stateChannel <-chan openvpn.State, sessionID session.SessionID) {
	for state := range stateChannel {
		manager.onStateChanged(state, sessionID)
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.status = statusNotConnected()
	log.Debug(managerLogPrefix, "State updater stopped")
}

func (manager *connectionManager) onStateChanged(state openvpn.State, sessionID session.SessionID) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

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
