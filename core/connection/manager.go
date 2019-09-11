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
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
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

// Creator creates new connection by given options and uses state channel to report state changes
type Creator func(serviceType string, stateChannel StateChannel, statisticsChannel StatisticsChannel) (Connection, error)

// SessionInfo contains all the relevant info of the current session
type SessionInfo struct {
	SessionID   session.ID
	ConsumerID  identity.Identity
	Proposal    market.ServiceProposal
	acknowledge func()
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

// PaymentIssuerFactory creates a new payment issuer from the given params
type PaymentIssuerFactory func(
	initialState promise.PaymentInfo,
	paymentDefinition dto.PaymentPerTime,
	messageChan chan balance.Message,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (PaymentIssuer, error)

// PaymentEngineFactory creates a new payment issuer from the given params
type PaymentEngineFactory func(invoice chan crypto.Invoice, dialog communication.Dialog, consumer identity.Identity) (PaymentIssuer, error)

type connectionManager struct {
	//these are passed on creation
	newDialog            DialogCreator
	paymentIssuerFactory PaymentIssuerFactory
	paymentEngineFactory PaymentEngineFactory
	newConnection        Creator
	eventPublisher       Publisher

	//these are populated by Connect at runtime
	ctx         context.Context
	status      Status
	statusLock  sync.RWMutex
	sessionInfo SessionInfo
	cleanup     []func() error
	cancel      func()

	discoLock sync.Mutex
}

// NewManager creates connection manager with given dependencies
func NewManager(
	dialogCreator DialogCreator,
	paymentIssuerFactory PaymentIssuerFactory,
	paymentEngineFactory PaymentEngineFactory,
	connectionCreator Creator,
	eventPublisher Publisher,
) *connectionManager {
	return &connectionManager{
		newDialog:            dialogCreator,
		paymentIssuerFactory: paymentIssuerFactory,
		newConnection:        connectionCreator,
		status:               statusNotConnected(),
		eventPublisher:       eventPublisher,
		paymentEngineFactory: paymentEngineFactory,
		cleanup:              make([]func() error, 0),
	}
}

func (manager *connectionManager) Connect(consumerID identity.Identity, proposal market.ServiceProposal, params ConnectParams) (err error) {
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

	sessionDTO, paymentInfo, err := manager.createSession(connection, dialog, consumerID, proposal)
	if err != nil {
		return err
	}

	err = manager.launchPayments(paymentInfo, dialog, consumerID, providerID)
	if err != nil {
		return err
	}

	err = manager.startConnection(connection, consumerID, proposal, params, sessionDTO, stateChannel, statisticsChannel)
	if err == context.Canceled {
		return ErrConnectionCancelled
	}
	return err
}

func (manager *connectionManager) launchPayments(paymentInfo *promise.PaymentInfo, dialog communication.Dialog, consumerID, providerID identity.Identity) error {
	var promiseState promise.PaymentInfo
	var useNewPayments bool
	if paymentInfo != nil {
		promiseState.FreeCredit = paymentInfo.FreeCredit
		promiseState.LastPromise = paymentInfo.LastPromise

		// if the server indicates that it will launch the new payments, so should we
		if paymentInfo.Supports == string(session.PaymentVersionV2) {
			useNewPayments = true
		}
	}

	// TODO: set the time and proper payment info
	payment := dto.PaymentPerTime{
		Price: money.Money{
			Currency: money.CurrencyMyst,
			Amount:   uint64(0),
		},
		Duration: time.Minute,
	}

	var payments PaymentIssuer
	if useNewPayments {
		log.Info("using new payments")
		invoices := make(chan crypto.Invoice)
		p, err := manager.paymentEngineFactory(invoices, dialog, consumerID)
		if err != nil {
			return err
		}
		payments = p
	} else {
		log.Info("using old payments")
		messageChan := make(chan balance.Message, 1)
		p, err := manager.paymentIssuerFactory(promiseState, payment, messageChan, dialog, consumerID, providerID)
		if err != nil {
			return err
		}
		payments = p
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
			log.Warn("cleanup error:", err)
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

func (manager *connectionManager) createSession(c Connection, dialog communication.Dialog, consumerID identity.Identity, proposal market.ServiceProposal) (session.SessionDto, *promise.PaymentInfo, error) {
	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return session.SessionDto{}, nil, err
	}

	consumerInfo := session.ConsumerInfo{
		// TODO: once we're supporting payments from another identity make the changes accordingly
		IssuerID: consumerID,
		Supports: session.PaymentVersionV2,
	}

	s, paymentInfo, err := session.RequestSessionCreate(dialog, proposal.ID, sessionCreateConfig, consumerInfo)
	if err != nil {
		return session.SessionDto{}, nil, err
	}

	manager.cleanup = append(manager.cleanup, func() error { return session.RequestSessionDestroy(dialog, s.ID) })

	// set the session info for future use
	manager.sessionInfo = SessionInfo{
		SessionID:  s.ID,
		ConsumerID: consumerID,
		Proposal:   proposal,
		acknowledge: func() {
			err := session.AcknowledgeSession(dialog, string(s.ID))
			if err != nil {
				log.Warn("acknowledge failed", err)
			}
		},
	}

	manager.eventPublisher.Publish(SessionEventTopic, SessionEvent{
		Status:      SessionCreatedStatus,
		SessionInfo: manager.sessionInfo,
	})

	manager.cleanup = append(manager.cleanup, func() error {
		manager.eventPublisher.Publish(SessionEventTopic, SessionEvent{
			Status:      SessionEndedStatus,
			SessionInfo: manager.sessionInfo,
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
			log.Info("cancelling connection initiation: ", err)
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

	//consume statistics right after start - openvpn3 will publish them even before connected state
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

	manager.eventPublisher.Publish(StateEventTopic, StateEvent{
		State:       NotConnected,
		SessionInfo: manager.sessionInfo,
	})
	return nil
}

func (manager *connectionManager) payForService(payments PaymentIssuer) {
	err := payments.Start()
	if err != nil {
		log.Error("payment error: ", err)
		err = manager.Disconnect()
		if err != nil {
			log.Error("could not disconnect gracefully:", err)
		}
	}
}

func (manager *connectionManager) connectionWaiter(connection Connection) {
	err := connection.Wait()
	if err != nil {
		log.Warn("connection exited with error: ", err)
	} else {
		log.Info("connection exited")
	}

	logDisconnectError(manager.Disconnect())
}

func (manager *connectionManager) waitForConnectedState(stateChannel <-chan State, sessionID session.ID) error {
	log.Trace("waiting for connected state")
	for {
		select {
		case state, more := <-stateChannel:
			if !more {
				return ErrConnectionFailed
			}

			switch state {
			case Connected:
				log.Trace("connected started event received")
				go manager.sessionInfo.acknowledge()
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

	log.Debug("state updater stopCalled")
	logDisconnectError(manager.Disconnect())
}

func (manager *connectionManager) consumeStats(statisticsChannel <-chan consumer.SessionStatistics) {
	for stats := range statisticsChannel {
		manager.eventPublisher.Publish(StatisticsEventTopic, stats)
	}
}

func (manager *connectionManager) onStateChanged(state State) {
	log.Trace("onStateChanged called")
	manager.eventPublisher.Publish(StateEventTopic, StateEvent{
		State:       state,
		SessionInfo: manager.sessionInfo,
	})

	switch state {
	case Connected:
		log.Trace("connected state issued")
		manager.setStatus(statusConnected(manager.sessionInfo.SessionID, manager.sessionInfo.Proposal))
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

func logDisconnectError(err error) {
	if err != nil && err != ErrNoConnection {
		log.Error("disconnect error", err)
	}
}
