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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/connectivity"
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
	// ErrInsufficientBalance indicates consumer has insufficient balance to connect to selected proposal
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrUnlockRequired indicates that the consumer identity has not been unlocked yet
	ErrUnlockRequired = errors.New("unlock required")
)

// IPCheckConfig contains common params for connection ip check.
type IPCheckConfig struct {
	MaxAttempts             int
	SleepDurationAfterCheck time.Duration
}

// KeepAliveConfig contains keep alive options.
type KeepAliveConfig struct {
	SendInterval    time.Duration
	MaxSendErrCount int
}

// Config contains common configuration options for connection manager.
type Config struct {
	IPCheck   IPCheckConfig
	KeepAlive KeepAliveConfig
}

// DefaultConfig returns default params.
func DefaultConfig() Config {
	return Config{
		IPCheck: IPCheckConfig{
			MaxAttempts:             6,
			SleepDurationAfterCheck: 3 * time.Second,
		},
		KeepAlive: KeepAliveConfig{
			SendInterval:    20 * time.Second,
			MaxSendErrCount: 5,
		},
	}
}

// Creator creates new connection by given options and uses state channel to report state changes
type Creator func(serviceType string) (Connection, error)

// SessionInfo contains all the relevant info of the current session
type SessionInfo struct {
	SessionID  session.ID
	ConsumerID identity.Identity
	Proposal   market.ServiceProposal
	ack        func()
}

// Acknowledge calls ack if it's set
func (s SessionInfo) Acknowledge() {
	if s.ack != nil {
		s.ack()
	}
}

// IsActive checks if session is active
func (s *SessionInfo) IsActive() bool {
	return s.SessionID != ""
}

// PaymentIssuer handles the payments for service
type PaymentIssuer interface {
	Start() error
	Stop()
}

type validator interface {
	Validate(consumerID identity.Identity, proposal market.ServiceProposal) error
}

// PaymentEngineFactory creates a new payment issuer from the given params
type PaymentEngineFactory func(paymentInfo session.PaymentInfo,
	dialog communication.Dialog, channel p2p.Channel,
	consumer, provider, accountant identity.Identity, proposal market.ServiceProposal, sessionID string) (PaymentIssuer, error)

type connectionManager struct {
	// These are passed on creation.
	newDialog                DialogCreator
	paymentEngineFactory     PaymentEngineFactory
	newConnection            Creator
	eventPublisher           eventbus.Publisher
	connectivityStatusSender connectivity.StatusSender
	ipResolver               ip.Resolver
	config                   Config
	statsReportInterval      time.Duration
	validator                validator
	p2pDialer                p2p.Dialer

	// These are populated by Connect at runtime.
	ctx                    context.Context
	status                 Status
	statusLock             sync.RWMutex
	sessionInfo            SessionInfo
	sessionInfoMu          sync.Mutex
	cleanup                []func() error
	cleanupAfterDisconnect []func() error
	cancel                 func()

	discoLock sync.Mutex
}

// NewManager creates connection manager with given dependencies
func NewManager(
	dialogCreator DialogCreator,
	paymentEngineFactory PaymentEngineFactory,
	connectionCreator Creator,
	eventPublisher eventbus.Publisher,
	connectivityStatusSender connectivity.StatusSender,
	ipResolver ip.Resolver,
	config Config,
	statsReportInterval time.Duration,
	validator validator,
	p2pDialer p2p.Dialer,
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
		config:                   config,
		statsReportInterval:      statsReportInterval,
		validator:                validator,
		p2pDialer:                p2pDialer,
	}
}

func (manager *connectionManager) Connect(consumerID, accountantID identity.Identity, proposal market.ServiceProposal, params ConnectParams) (err error) {
	if manager.Status().State != NotConnected {
		return ErrAlreadyExists
	}

	err = manager.validator.Validate(consumerID, proposal)
	if err != nil {
		return err
	}

	manager.ctx, manager.cancel = context.WithCancel(context.Background())

	manager.publishStateEvent(Connecting)
	manager.setStatus(statusConnecting())
	defer func() {
		if err != nil {
			manager.setStatus(statusNotConnected())
			manager.publishStateEvent(NotConnected)
		}
	}()

	providerID := identity.FromAddress(proposal.ProviderID)

	channel := manager.createP2PChannel(consumerID, providerID, proposal)

	var dialog communication.Dialog
	if channel == nil {
		dialog, err = manager.createDialog(consumerID, providerID, proposal.ProviderContacts[0])
		if err != nil {
			return err
		}
	}

	connection, err := manager.newConnection(proposal.ServiceType)
	if err != nil {
		return err
	}

	var paymentInfo session.PaymentInfo
	var sessionDTO session.SessionDto

	if channel != nil {
		sessionDTO, paymentInfo, err = manager.createP2PSession(connection, channel, consumerID, accountantID, proposal)
	} else {
		sessionDTO, paymentInfo, err = manager.createSession(connection, dialog, consumerID, accountantID, proposal)
	}
	if err != nil {
		manager.sendSessionStatus(dialog, channel, consumerID, "", connectivity.StatusSessionEstablishmentFailed, err)
		return err
	}

	err = manager.launchPayments(paymentInfo, dialog, channel, consumerID, providerID, accountantID, proposal, sessionDTO.ID)
	if err != nil {
		manager.sendSessionStatus(dialog, channel, consumerID, sessionDTO.ID, connectivity.StatusSessionPaymentsFailed, err)
		return err
	}

	originalPublicIP := manager.getPublicIP()
	// Try to establish connection with peer.
	err = manager.startConnection(connection, consumerID, proposal, params, sessionDTO, channel)
	if err != nil {
		if err == context.Canceled {
			return ErrConnectionCancelled
		}

		manager.cleanupAfterDisconnect = append(manager.cleanupAfterDisconnect, func() error {
			return manager.sendSessionStatus(dialog, channel, consumerID, sessionDTO.ID, connectivity.StatusConnectionFailed, err)
		})
		manager.publishStateEvent(StateConnectionFailed)

		log.Info().Err(err).Msg("Cancelling connection initiation: ")
		manager.Cancel()
		return err
	}

	go manager.keepAliveLoop(channel, sessionDTO.ID)
	go manager.checkSessionIP(dialog, channel, consumerID, sessionDTO.ID, originalPublicIP)

	return nil
}

// checkSessionIP checks if IP has changed after connection was established.
func (manager *connectionManager) checkSessionIP(dialog communication.Dialog, channel p2p.Channel, consumerID identity.Identity, sessionID session.ID, originalPublicIP string) {
	for i := 1; i <= manager.config.IPCheck.MaxAttempts; i++ {
		// Skip check if not connected. This may happen when context was canceled via Disconnect.
		if manager.Status().State != Connected {
			return
		}

		newPublicIP := manager.getPublicIP()

		// If ip is changed notify peer that connection is successful.
		if originalPublicIP != newPublicIP {
			manager.sendSessionStatus(dialog, channel, consumerID, sessionID, connectivity.StatusConnectionOk, nil)
			return
		}

		// Notify peer and quality oracle that ip is not changed after tunnel connection was established.
		if i == manager.config.IPCheck.MaxAttempts {
			manager.sendSessionStatus(dialog, channel, consumerID, sessionID, connectivity.StatusSessionIPNotChanged, nil)
			manager.publishStateEvent(StateIPNotChanged)
			return
		}

		time.Sleep(manager.config.IPCheck.SleepDurationAfterCheck)
	}
}

// sendSessionStatus sends session connectivity status to other peer.
func (manager *connectionManager) sendSessionStatus(dialog communication.Dialog, channel p2p.ChannelSender, consumerID identity.Identity, sessionID session.ID, code connectivity.StatusCode, errDetails error) error {
	var errDetailsMsg string
	if errDetails != nil {
		errDetailsMsg = errDetails.Error()
	}

	if channel == nil {
		return manager.connectivityStatusSender.Send(dialog, &connectivity.StatusMessage{
			SessionID:  string(sessionID),
			StatusCode: code,
			Message:    errDetailsMsg,
		})
	}

	sessionStatus := &pb.SessionStatus{
		ConsumerID: consumerID.Address,
		SessionID:  string(sessionID),
		Code:       uint32(code),
		Message:    errDetailsMsg,
	}

	log.Debug().Msgf("Sending session status P2P message to %q: %s", p2p.TopicSessionStatus, sessionStatus.String())

	_, err := channel.Send(p2p.TopicSessionStatus, p2p.ProtoMessage(sessionStatus))
	if err != nil {
		return fmt.Errorf("could not send p2p session status message: %w", err)
	}

	return nil
}

func (manager *connectionManager) getPublicIP() string {
	currentPublicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		log.Error().Err(err).Msg("Could not get current public IP")
		return ""
	}
	return currentPublicIP
}

func (manager *connectionManager) launchPayments(paymentInfo session.PaymentInfo, dialog communication.Dialog, channel p2p.Channel, consumerID, providerID, accountantID identity.Identity, proposal market.ServiceProposal, sessionID session.ID) error {
	payments, err := manager.paymentEngineFactory(paymentInfo, dialog, channel, consumerID, providerID, accountantID, proposal, string(sessionID))
	if err != nil {
		return err
	}
	manager.cleanup = append(manager.cleanup, func() error {
		log.Trace().Msg("Cleaning: payments")
		defer log.Trace().Msg("Cleaning: payments DONE")
		payments.Stop()
		return nil
	})

	go manager.payForService(payments)
	return nil
}

func (manager *connectionManager) cleanConnection() {
	manager.cancel()
	for i := len(manager.cleanup) - 1; i >= 0; i-- {
		log.Trace().Msgf("Connection cleaning up: (%v/%v)", i+1, len(manager.cleanup))
		err := manager.cleanup[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	manager.cleanup = nil
}

func (manager *connectionManager) cleanAfterDisconnect() {
	manager.cancel()
	for i := len(manager.cleanupAfterDisconnect) - 1; i >= 0; i-- {
		log.Trace().Msgf("Connection cleaning up (after disconnect): (%v/%v)", i+1, len(manager.cleanupAfterDisconnect))
		err := manager.cleanupAfterDisconnect[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	manager.cleanupAfterDisconnect = nil
}

func (manager *connectionManager) createDialog(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error) {
	dialog, err := manager.newDialog(consumerID, providerID, contact)
	if err != nil {
		return nil, err
	}

	manager.cleanupAfterDisconnect = append(manager.cleanupAfterDisconnect, func() error {
		log.Trace().Msg("Cleaning: closing dialog")
		defer log.Trace().Msg("Cleaning: closing dialog DONE")
		return dialog.Close()
	})
	return dialog, err
}

func (manager *connectionManager) createP2PChannel(consumerID, providerID identity.Identity, proposal market.ServiceProposal) p2p.Channel {
	channel, err := manager.p2pDialer.Dial(consumerID, providerID, proposal.ServiceType, 10*time.Second)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to establish p2p channel")
	} else {
		manager.cleanupAfterDisconnect = append(manager.cleanupAfterDisconnect, func() error {
			log.Trace().Msg("Cleaning: closing P2P communication channel")
			defer log.Trace().Msg("Cleaning: P2P communication channel DONE")

			return channel.Close()
		})
	}
	return channel
}

func (manager *connectionManager) createP2PSession(c Connection, p2pChannel p2p.ChannelSender, consumerID, accountantID identity.Identity, proposal market.ServiceProposal) (session.SessionDto, session.PaymentInfo, error) {
	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return session.SessionDto{}, session.PaymentInfo{}, fmt.Errorf("could not get session config: %w", err)
	}

	config, err := json.Marshal(sessionCreateConfig)
	if err != nil {
		return session.SessionDto{}, session.PaymentInfo{}, fmt.Errorf("could not marshal session config: %w", err)
	}

	sessionRequest := &pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:             consumerID.Address,
			AccountantID:   accountantID.Address,
			PaymentVersion: string(session.PaymentVersionV3),
		},
		ProposalID: int64(proposal.ID),
		Config:     config,
	}
	log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionCreate, sessionRequest.String())
	res, err := p2pChannel.Send(p2p.TopicSessionCreate, p2p.ProtoMessage(sessionRequest))
	if err != nil {
		return session.SessionDto{}, session.PaymentInfo{}, fmt.Errorf("could not send p2p session create request: %w", err)
	}

	var sessionResponce pb.SessionResponse
	err = res.UnmarshalProto(&sessionResponce)
	if err != nil {
		return session.SessionDto{}, session.PaymentInfo{}, fmt.Errorf("could not unmarshal session reply to proto: %w", err)
	}

	sessionID := session.ID(sessionResponce.GetID())
	manager.cleanupAfterDisconnect = append(manager.cleanupAfterDisconnect, func() error {
		log.Trace().Msg("Cleaning: requesting session destroy")
		defer log.Trace().Msg("Cleaning: requesting session destroy DONE")

		sessionDestroy := &pb.SessionInfo{
			ConsumerID: consumerID.Address,
			SessionID:  sessionResponce.GetID(),
		}

		log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionDestroy, sessionDestroy.String())
		_, err := p2pChannel.Send(p2p.TopicSessionDestroy, p2p.ProtoMessage(sessionDestroy))
		if err != nil {
			return fmt.Errorf("could not send session destroy request: %w", err)
		}

		return nil
	})

	manager.saveSessionInfo(SessionInfo{
		SessionID:  sessionID,
		ConsumerID: consumerID,
		Proposal:   proposal,
		ack: func() {
			pc := &pb.SessionInfo{
				ConsumerID: consumerID.Address,
				SessionID:  string(sessionID),
			}
			log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionAcknowledge, pc.String())
			_, err := p2pChannel.Send(p2p.TopicSessionAcknowledge, p2p.ProtoMessage(pc))
			if err != nil {
				log.Warn().Err(err).Msg("Acknowledge failed")
			}
		},
	})

	return session.SessionDto{
		ID:     sessionID,
		Config: sessionResponce.GetConfig(),
	}, session.PaymentInfo{Supports: sessionResponce.GetPaymentInfo()}, nil
}

func (manager *connectionManager) createSession(c Connection, dialog communication.Dialog, consumerID, accountantID identity.Identity, proposal market.ServiceProposal) (session.SessionDto, session.PaymentInfo, error) {
	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return session.SessionDto{}, session.PaymentInfo{}, err
	}

	consumerInfo := session.ConsumerInfo{
		IssuerID:       consumerID,
		AccountantID:   accountantID,
		PaymentVersion: session.PaymentVersionV3,
	}

	s, paymentInfo, err := session.RequestSessionCreate(dialog, proposal.ID, sessionCreateConfig, consumerInfo)
	if err != nil {
		return session.SessionDto{}, session.PaymentInfo{}, err
	}

	manager.cleanupAfterDisconnect = append(manager.cleanupAfterDisconnect, func() error {
		log.Trace().Msg("Cleaning: requesting session destroy")
		defer log.Trace().Msg("Cleaning: requesting session destroy DONE")
		return session.RequestSessionDestroy(dialog, s.ID)
	})

	manager.saveSessionInfo(SessionInfo{
		SessionID:  s.ID,
		ConsumerID: consumerID,
		Proposal:   proposal,
		ack: func() {
			err := session.AcknowledgeSession(dialog, string(s.ID))
			if err != nil {
				log.Warn().Err(err).Msg("Acknowledge failed")
			}
		},
	})

	return s, paymentInfo, nil
}

func (manager *connectionManager) saveSessionInfo(sessionInfo SessionInfo) {
	manager.setCurrentSession(sessionInfo)

	manager.eventPublisher.Publish(AppTopicConsumerSession, SessionEvent{
		Status:      SessionCreatedStatus,
		SessionInfo: manager.getCurrentSession(),
	})

	manager.cleanup = append(manager.cleanup, func() error {
		log.Trace().Msg("Cleaning: publishing session ended status")
		defer log.Trace().Msg("Cleaning: publishing session ended status DONE")
		manager.eventPublisher.Publish(AppTopicConsumerSession, SessionEvent{
			Status:      SessionEndedStatus,
			SessionInfo: manager.getCurrentSession(),
		})
		manager.setCurrentSession(SessionInfo{})
		return nil
	})
}

func (manager *connectionManager) startConnection(
	conn Connection,
	consumerID identity.Identity,
	proposal market.ServiceProposal,
	params ConnectParams,
	sessionDTO session.SessionDto,
	channel p2p.Channel,
) (err error) {
	connectOptions := ConnectOptions{
		SessionID:     sessionDTO.ID,
		SessionConfig: sessionDTO.Config,
		DNS:           params.DNS,
		ConsumerID:    consumerID,
		ProviderID:    identity.FromAddress(proposal.ProviderID),
		Proposal:      proposal,
	}

	if channel != nil {
		connectOptions.ProviderNATConn = channel.ServiceConn()
		connectOptions.ChannelConn = channel.Conn()
	}

	if err = conn.Start(connectOptions); err != nil {
		return err
	}

	statsPublisher := newStatsPublisher(manager.eventPublisher, manager.statsReportInterval)
	go statsPublisher.start(manager.getCurrentSession(), conn)

	manager.cleanup = append(manager.cleanup, func() error {
		log.Trace().Msg("Cleaning: stopping statistics publisher")
		defer log.Trace().Msg("Cleaning: stopping statistics publisher DONE")
		statsPublisher.stop()
		return nil
	})
	manager.cleanup = append(manager.cleanup, func() error {
		log.Trace().Msg("Cleaning: stopping connection")
		defer log.Trace().Msg("Cleaning: stopping connection DONE")
		conn.Stop()
		return nil
	})

	err = manager.setupTrafficBlock(params.DisableKillSwitch)
	if err != nil {
		return err
	}

	err = manager.waitForConnectedState(conn.State())
	if err != nil {
		return err
	}

	go manager.consumeConnectionStates(conn.State())
	go manager.connectionWaiter(conn)
	return nil
}

func (manager *connectionManager) Status() Status {
	manager.statusLock.RLock()
	defer manager.statusLock.RUnlock()

	return manager.status
}

func (manager *connectionManager) setStatus(cs Status) {
	manager.statusLock.Lock()
	log.Info().Msgf("Connection state: %v → %v", manager.status.State, cs.State)
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

	manager.cleanAfterDisconnect()

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

func (manager *connectionManager) waitForConnectedState(stateChannel <-chan State) error {
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
				go manager.getCurrentSession().Acknowledge()
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

func (manager *connectionManager) onStateChanged(state State) {
	log.Debug().Msg("onStateChanged called")
	manager.publishStateEvent(state)

	switch state {
	case Connected:
		sessionInfo := manager.getCurrentSession()
		manager.setStatus(statusConnected(sessionInfo.SessionID, sessionInfo.Proposal, sessionInfo.ConsumerID))
	case Reconnecting:
		manager.setStatus(statusReconnecting())
	}
}

func (manager *connectionManager) setupTrafficBlock(disableKillSwitch bool) error {
	if disableKillSwitch {
		return nil
	}

	outboundIP, err := manager.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return err
	}

	removeRule, err := firewall.BlockNonTunnelTraffic(firewall.Session, outboundIP)
	if err != nil {
		return err
	}
	manager.cleanup = append(manager.cleanup, func() error {
		log.Trace().Msg("Cleaning: traffic block rule")
		defer log.Trace().Msg("Cleaning: traffic block rule DONE")
		removeRule()
		return nil
	})
	return nil
}

func (manager *connectionManager) publishStateEvent(state State) {
	manager.eventPublisher.Publish(AppTopicConsumerConnectionState, StateEvent{
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

func (manager *connectionManager) keepAliveLoop(channel p2p.Channel, sessionID session.ID) {
	// TODO: Remove this check once all provider migrates to p2p.
	if channel == nil {
		return
	}

	// Register handler for handling p2p keep alive pings from provider.
	channel.Handle(p2p.TopicKeepAlive, func(c p2p.Context) error {
		var ping pb.P2PKeepAlivePing
		if err := c.Request().UnmarshalProto(&ping); err != nil {
			return err
		}

		log.Debug().Msgf("Received p2p keepalive ping with SessionID=%s", ping.SessionID)
		return c.OK()
	})

	// Send pings to provider.
	var errCount int
	for {
		select {
		case <-manager.ctx.Done():
			return
		case <-time.After(manager.config.KeepAlive.SendInterval):
			sess := manager.getCurrentSession()
			msg := &pb.P2PKeepAlivePing{
				SessionID: string(sessionID),
			}
			_, err := channel.Send(p2p.TopicKeepAlive, p2p.ProtoMessage(msg))
			if err != nil {
				log.Err(err).Msgf("Failed to send p2p keepalive ping. SessionID=%s", sess.SessionID)
				errCount++
				if errCount == manager.config.KeepAlive.MaxSendErrCount {
					log.Error().Msgf("Max p2p keepalive err count reached, disconnecting. SessionID=%s", sess.SessionID)
					manager.Disconnect()
					return
				}
			} else {
				errCount = 0
			}
		}
	}
}

func logDisconnectError(err error) {
	if err != nil && err != ErrNoConnection {
		log.Error().Err(err).Msg("Disconnect error")
	}
}
