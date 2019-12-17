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
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

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

// IPCheckParams contains common params for connection ip check.
type IPCheckParams struct {
	MaxAttempts             int
	SleepDurationAfterCheck time.Duration
	Done                    chan struct{}
}

// DefaultIPCheckParams returns default params.
func DefaultIPCheckParams() IPCheckParams {
	return IPCheckParams{
		MaxAttempts:             6,
		SleepDurationAfterCheck: 3 * time.Second,
		Done:                    make(chan struct{}, 1),
	}
}

// Creator creates new connection by given options and uses state channel to report state changes
type Creator func(serviceType string, stateChannel StateChannel, statisticsChannel StatisticsChannel) (Connection, error)

// SessionInfo contains all the relevant info of the current session
type SessionInfo struct {
	SessionID   session.ID
	ConsumerID  identity.Identity
	Proposal    market.ServiceProposal
	acknowledge func()
}

// IsActive checks if session is active
func (s *SessionInfo) IsActive() bool {
	return s.SessionID != ""
}

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, data interface{})
}

// PaymentIssuer handles the payments for service
type PaymentIssuer interface {
	Start() error
	Stop()
}

// PaymentEngineFactory creates a new payment issuer from the given params
type PaymentEngineFactory func(paymentInfo *promise.PaymentInfo,
	dialog communication.Dialog,
	consumer, provider, accountant identity.Identity) (PaymentIssuer, error)

type connectionManager struct {
	// These are passed on creation.
	newDialog                DialogCreator
	paymentEngineFactory     PaymentEngineFactory
	newConnection            Creator
	eventPublisher           Publisher
	connectivityStatusSender connectivity.StatusSender
	ipResolver               ip.Resolver
	ipCheckParams            IPCheckParams

	// These are populated by Connect at runtime.
	ctx             context.Context
	status          Status
	statusLock      sync.RWMutex
	sessionInfo     SessionInfo
	disablePayments bool
	sessionInfoMu   sync.Mutex
	cleanup         []func() error
	cancel          func()

	discoLock sync.Mutex
}

// NewManager creates connection manager with given dependencies
func NewManager(
	dialogCreator DialogCreator,
	paymentEngineFactory PaymentEngineFactory,
	connectionCreator Creator,
	eventPublisher Publisher,
	connectivityStatusSender connectivity.StatusSender,
	ipResolver ip.Resolver,
	ipCheckParams IPCheckParams,
	disablePayments bool,
) *connectionManager {
	return &connectionManager{
		newDialog:                dialogCreator,
		newConnection:            connectionCreator,
		status:                   statusNotConnected(),
		eventPublisher:           eventPublisher,
		paymentEngineFactory:     paymentEngineFactory,
		connectivityStatusSender: connectivityStatusSender,
		cleanup:                  make([]func() error, 0),
		ipResolver:               ipResolver,
		ipCheckParams:            ipCheckParams,
		disablePayments:          disablePayments,
	}
}

func (manager *connectionManager) Connect(consumerID, accountantID identity.Identity, proposal market.ServiceProposal, params ConnectParams) (err error) {
	if manager.Status().State != NotConnected {
		return ErrAlreadyExists
	}

	manager.ctx, manager.cancel = context.WithCancel(context.Background())

	manager.setStatus(statusConnecting())
	defer func() {
		if err != nil {
			manager.setStatus(statusNotConnected())
		}
	}()

	providerID := identity.FromAddress(proposal.ProviderID)
	dialog, err := manager.createDialog(consumerID, providerID, proposal.ProviderContacts[0])
	if err != nil {
		return err
	}

	stateChannel := make(chan State, 10)
	statisticsChannel := make(chan consumer.SessionStatistics, 10)

	connection, err := manager.newConnection(proposal.ServiceType, stateChannel, statisticsChannel)
	if err != nil {
		return err
	}

	sessionDTO, paymentInfo, err := manager.createSession(connection, dialog, consumerID, accountantID, proposal)
	if err != nil {
		manager.sendSessionStatus(dialog, "", connectivity.StatusSessionEstablishmentFailed, err)
		return err
	}

	err = manager.launchPayments(paymentInfo, dialog, consumerID, providerID, accountantID)
	if err != nil {
		manager.sendSessionStatus(dialog, sessionDTO.ID, connectivity.StatusSessionPaymentsFailed, err)
		return err
	}

	originalPublicIP := manager.getPublicIP()
	// Try to establish connection with peer.
	err = manager.startConnection(connection, consumerID, proposal, params, sessionDTO, stateChannel, statisticsChannel)
	if err != nil {
		if err == context.Canceled {
			return ErrConnectionCancelled
		}
		manager.sendSessionStatus(dialog, sessionDTO.ID, connectivity.StatusConnectionFailed, err)
		manager.publishStateEvent(StateConnectionFailed)
		return err
	}

	go manager.checkSessionIP(dialog, sessionDTO.ID, originalPublicIP)
	return nil
}

// checkSessionIP checks if IP has changed after connection was established.
func (manager *connectionManager) checkSessionIP(dialog communication.Dialog, sessionID session.ID, originalPublicIP string) {
	defer func() {
		// Notify that check is done.
		manager.ipCheckParams.Done <- struct{}{}
	}()

	for i := 1; i <= manager.ipCheckParams.MaxAttempts; i++ {
		// Skip check if not connected. This may happen when context was canceled via Disconnect.
		if manager.Status().State != Connected {
			return
		}

		newPublicIP := manager.getPublicIP()

		// If ip is changed notify peer that connection is successful.
		if originalPublicIP != newPublicIP {
			manager.sendSessionStatus(dialog, sessionID, connectivity.StatusConnectionOk, nil)
			return
		}

		// Notify peer and quality oracle that ip is not changed after tunnel connection was established.
		if i == manager.ipCheckParams.MaxAttempts {
			manager.sendSessionStatus(dialog, sessionID, connectivity.StatusSessionIPNotChanged, nil)
			manager.publishStateEvent(StateIPNotChanged)
			return
		}

		time.Sleep(manager.ipCheckParams.SleepDurationAfterCheck)
	}
}

// sendSessionStatus sends session connectivity status to other peer.
func (manager *connectionManager) sendSessionStatus(dialog communication.Dialog, sessionID session.ID, code connectivity.StatusCode, errDetails error) {
	var errDetailsMsg string
	if errDetails != nil {
		errDetailsMsg = errDetails.Error()
	}
	manager.connectivityStatusSender.Send(dialog, &connectivity.StatusMessage{
		SessionID:  string(sessionID),
		StatusCode: code,
		Message:    errDetailsMsg,
	})
}

func (manager *connectionManager) getPublicIP() string {
	currentPublicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		log.Error().Err(err).Msg("Could not get current public IP")
		return ""
	}
	return currentPublicIP
}

func (manager *connectionManager) launchPayments(paymentInfo *promise.PaymentInfo, dialog communication.Dialog, consumerID, providerID, accountantID identity.Identity) error {
	payments, err := manager.paymentEngineFactory(paymentInfo, dialog, consumerID, providerID, accountantID)
	if err != nil {
		return err
	}
	manager.cleanup = append(manager.cleanup, func() error {
		payments.Stop()
		return nil
	})

	go manager.payForService(payments)
	return nil
}

func (manager *connectionManager) cleanConnection() {
	manager.cancel()
	for i := len(manager.cleanup) - 1; i >= 0; i-- {
		err := manager.cleanup[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	manager.cleanup = make([]func() error, 0)
}

func (manager *connectionManager) createDialog(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error) {
	dialog, err := manager.newDialog(consumerID, providerID, contact)
	if err != nil {
		return nil, err
	}

	manager.cleanup = append(manager.cleanup, dialog.Close)
	return dialog, err
}

func (manager *connectionManager) createSession(c Connection, dialog communication.Dialog, consumerID, accountantID identity.Identity, proposal market.ServiceProposal) (session.SessionDto, *promise.PaymentInfo, error) {
	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return session.SessionDto{}, nil, err
	}

	paymentVersion := session.PaymentVersionV3
	if manager.disablePayments {
		paymentVersion = "legacy"
	}
	consumerInfo := session.ConsumerInfo{
		// TODO: once we're supporting payments from another identity make the changes accordingly
		IssuerID:       consumerID,
		AccountantID:   accountantID,
		PaymentVersion: paymentVersion,
	}

	s, paymentInfo, err := session.RequestSessionCreate(dialog, proposal.ID, sessionCreateConfig, consumerInfo)
	if err != nil {
		return session.SessionDto{}, nil, err
	}

	manager.cleanup = append(manager.cleanup, func() error { return session.RequestSessionDestroy(dialog, s.ID) })

	// set the session info for future use
	sessionInfo := SessionInfo{
		SessionID:  s.ID,
		ConsumerID: consumerID,
		Proposal:   proposal,
		acknowledge: func() {
			err := session.AcknowledgeSession(dialog, string(s.ID))
			if err != nil {
				log.Warn().Err(err).Msg("Acknowledge failed")
			}
		},
	}
	manager.setCurrentSession(sessionInfo)

	manager.eventPublisher.Publish(SessionEventTopic, SessionEvent{
		Status:      SessionCreatedStatus,
		SessionInfo: manager.getCurrentSession(),
	})

	manager.cleanup = append(manager.cleanup, func() error {
		manager.eventPublisher.Publish(SessionEventTopic, SessionEvent{
			Status:      SessionEndedStatus,
			SessionInfo: manager.getCurrentSession(),
		})
		return nil
	})

	return s, paymentInfo, nil
}

func (manager *connectionManager) startConnection(
	connection Connection,
	consumerID identity.Identity,
	proposal market.ServiceProposal,
	params ConnectParams,
	sessionDTO session.SessionDto,
	stateChannel chan State,
	statisticsChannel chan consumer.SessionStatistics) (err error) {

	defer func() {
		if err != nil {
			log.Info().Err(err).Msg("Cancelling connection initiation: ")
			manager.Cancel()
		}
	}()

	connectOptions := ConnectOptions{
		SessionID:     sessionDTO.ID,
		SessionConfig: sessionDTO.Config,
		EnableDNS:     params.EnableDNS,
		ConsumerID:    consumerID,
		ProviderID:    identity.FromAddress(proposal.ProviderID),
		Proposal:      proposal,
	}

	if err = connection.Start(connectOptions); err != nil {
		return err
	}
	manager.cleanup = append(manager.cleanup, func() error {
		connection.Stop()
		return nil
	})

	err = manager.setupTrafficBlock(params.DisableKillSwitch)
	if err != nil {
		return err
	}

	// Consume statistics right after start - openvpn3 will publish them even before connected state.
	go manager.consumeStats(statisticsChannel)
	err = manager.waitForConnectedState(stateChannel, sessionDTO.ID)
	if err != nil {
		return err
	}

	go manager.consumeConnectionStates(stateChannel)
	go manager.connectionWaiter(connection)
	return nil
}

func (manager *connectionManager) Status() Status {
	manager.statusLock.RLock()
	defer manager.statusLock.RUnlock()

	return manager.status
}

func (manager *connectionManager) setStatus(cs Status) {
	manager.statusLock.Lock()
	manager.status = cs
	manager.statusLock.Unlock()
}

func (manager *connectionManager) Cancel() {
	status := statusCanceled()
	manager.setStatus(status)
	manager.onStateChanged(status.State)
	logDisconnectError(manager.Disconnect())
}

func (manager *connectionManager) Disconnect() error {
	manager.discoLock.Lock()
	defer manager.discoLock.Unlock()

	if manager.Status().State == NotConnected {
		return ErrNoConnection
	}

	manager.setStatus(statusDisconnecting())
	manager.cleanConnection()
	manager.setStatus(statusNotConnected())

	manager.publishStateEvent(NotConnected)
	return nil
}

func (manager *connectionManager) payForService(payments PaymentIssuer) {
	err := payments.Start()
	if err != nil {
		log.Error().Err(err).Msg("Payment error")
		err = manager.Disconnect()
		if err != nil {
			log.Error().Err(err).Msg("Could not disconnect gracefully")
		}
	}
}

func (manager *connectionManager) connectionWaiter(connection Connection) {
	err := connection.Wait()
	if err != nil {
		log.Warn().Err(err).Msg("Connection exited with error")
	} else {
		log.Info().Msg("Connection exited")
	}

	logDisconnectError(manager.Disconnect())
}

func (manager *connectionManager) waitForConnectedState(stateChannel <-chan State, sessionID session.ID) error {
	log.Debug().Msg("waiting for connected state")
	for {
		select {
		case state, more := <-stateChannel:
			if !more {
				return ErrConnectionFailed
			}

			switch state {
			case Connected:
				log.Debug().Msg("Connected started event received")
				go manager.getCurrentSession().acknowledge()
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

	log.Debug().Msg("State updater stopCalled")
	logDisconnectError(manager.Disconnect())
}

func (manager *connectionManager) consumeStats(statisticsChannel <-chan consumer.SessionStatistics) {
	for stats := range statisticsChannel {
		manager.eventPublisher.Publish(StatisticsEventTopic, SessionStatsEvent{
			Stats:       stats,
			SessionInfo: manager.getCurrentSession(),
		})
	}
}

func (manager *connectionManager) onStateChanged(state State) {
	log.Debug().Msg("onStateChanged called")
	manager.publishStateEvent(state)

	switch state {
	case Connected:
		log.Debug().Msg("Connected state issued")
		sessionInfo := manager.getCurrentSession()
		manager.setStatus(statusConnected(sessionInfo.SessionID, sessionInfo.Proposal))
	case Reconnecting:
		manager.setStatus(statusReconnecting())
	}
}

func (manager *connectionManager) setupTrafficBlock(disableKillSwitch bool) error {
	if disableKillSwitch {
		return nil
	}

	removeRule, err := firewall.BlockNonTunnelTraffic(firewall.Session)
	if err != nil {
		return err
	}
	manager.cleanup = append(manager.cleanup, func() error {
		removeRule()
		return nil
	})
	return nil
}

func (manager *connectionManager) publishStateEvent(state State) {
	manager.eventPublisher.Publish(StateEventTopic, StateEvent{
		State:       state,
		SessionInfo: manager.getCurrentSession(),
	})
}

func (manager *connectionManager) setCurrentSession(info SessionInfo) {
	manager.sessionInfoMu.Lock()
	defer manager.sessionInfoMu.Unlock()

	manager.sessionInfo = info
}

func (manager *connectionManager) getCurrentSession() SessionInfo {
	manager.sessionInfoMu.Lock()
	defer manager.sessionInfoMu.Unlock()

	return manager.sessionInfo
}

func logDisconnectError(err error) {
	if err != nil && err != ErrNoConnection {
		log.Error().Err(err).Msg("Disconnect error")
	}
}
