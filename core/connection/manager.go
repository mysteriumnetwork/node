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
	stdErrors "errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
	SendTimeout     time.Duration
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
			SendTimeout:     5 * time.Second,
			MaxSendErrCount: 5,
		},
	}
}

// Creator creates new connection by given options and uses state channel to report state changes
type Creator func(serviceType string) (Connection, error)

// PaymentIssuer handles the payments for service
type PaymentIssuer interface {
	Start() error
	Stop()
}

type validator interface {
	Validate(consumerID identity.Identity, proposal market.ServiceProposal) error
}

// TimeGetter function returns current time
type TimeGetter func() time.Time

// PaymentEngineFactory creates a new payment issuer from the given params
type PaymentEngineFactory func(paymentInfo session.PaymentInfo,
	dialog communication.Dialog, channel p2p.Channel,
	consumer, provider identity.Identity, accountant common.Address, proposal market.ServiceProposal, sessionID string) (PaymentIssuer, error)

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
	timeGetter               TimeGetter

	// These are populated by Connect at runtime.
	ctx                    context.Context
	ctxLock                sync.RWMutex
	status                 Status
	statusLock             sync.RWMutex
	cleanupLock            sync.Mutex
	cleanup                []func() error
	cleanupAfterDisconnect []func() error
	acknowledge            func()
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
		status:                   Status{State: NotConnected},
		eventPublisher:           eventPublisher,
		paymentEngineFactory:     paymentEngineFactory,
		connectivityStatusSender: connectivityStatusSender,
		cleanup:                  make([]func() error, 0),
		ipResolver:               ipResolver,
		config:                   config,
		statsReportInterval:      statsReportInterval,
		validator:                validator,
		p2pDialer:                p2pDialer,
		timeGetter:               time.Now,
	}
}

func (m *connectionManager) Connect(consumerID identity.Identity, accountantID common.Address, proposal market.ServiceProposal, params ConnectParams) (err error) {
	if m.Status().State != NotConnected {
		return ErrAlreadyExists
	}

	err = m.validator.Validate(consumerID, proposal)
	if err != nil {
		return err
	}

	m.ctxLock.Lock()
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.ctxLock.Unlock()

	m.statusConnecting(consumerID, accountantID, proposal)
	defer func() {
		if err != nil {
			log.Err(err).Msg("Connect failed, disconnecting")
			m.disconnect()
		}
	}()

	providerID := identity.FromAddress(proposal.ProviderID)

	var channel p2p.Channel
	if contact, err := p2p.ParseContact(proposal.ProviderContacts); err == nil {
		channel, err = m.createP2PChannel(m.currentCtx(), consumerID, providerID, proposal.ServiceType, contact)
		if err != nil {
			return fmt.Errorf("could not create p2p channel: %w", err)
		}
	} else {
		if stdErrors.Is(err, p2p.ErrContactNotFound) {
			log.Debug().Msgf("Provider %s doesn't support p2p, will fallback to dialog", providerID.Address)
		} else {
			return err
		}
	}

	var dialog communication.Dialog
	if channel == nil {
		dialog, err = m.createDialog(consumerID, providerID, proposal.ProviderContacts[0])
		if err != nil {
			return err
		}
	}

	connection, err := m.newConnection(proposal.ServiceType)
	if err != nil {
		return err
	}

	var serviceConn, channelConn *net.UDPConn
	var sessionDTO session.CreateResponse

	if channel != nil {
		sessionDTO, err = m.createP2PSession(m.currentCtx(), connection, channel, consumerID, accountantID, proposal)
		serviceConn = channel.ServiceConn()
		channelConn = channel.Conn()
	} else {
		sessionDTO, err = m.createSession(connection, dialog, consumerID, accountantID, proposal)
	}
	if err != nil {
		m.sendSessionStatus(dialog, channel, consumerID, sessionDTO.Session.ID, connectivity.StatusSessionEstablishmentFailed, err)
		return err
	}

	err = m.launchPayments(sessionDTO.PaymentInfo, dialog, channel, consumerID, providerID, accountantID, proposal, sessionDTO.Session.ID)
	if err != nil {
		m.sendSessionStatus(dialog, channel, consumerID, sessionDTO.Session.ID, connectivity.StatusSessionPaymentsFailed, err)
		return err
	}

	originalPublicIP := m.getPublicIP()
	// Try to establish connection with peer.
	err = m.startConnection(m.currentCtx(), connection, params.DisableKillSwitch, ConnectOptions{
		SessionID:       sessionDTO.Session.ID,
		SessionConfig:   sessionDTO.Session.Config,
		DNS:             params.DNS,
		ConsumerID:      consumerID,
		ProviderID:      providerID,
		Proposal:        proposal,
		ProviderNATConn: serviceConn,
		ChannelConn:     channelConn,
	})
	if err != nil {
		if err == context.Canceled {
			return ErrConnectionCancelled
		}
		m.addCleanupAfterDisconnect(func() error {
			return m.sendSessionStatus(dialog, channel, consumerID, sessionDTO.Session.ID, connectivity.StatusConnectionFailed, err)
		})
		m.publishStateEvent(StateConnectionFailed)

		log.Info().Err(err).Msg("Cancelling connection initiation: ")
		m.Cancel()
		return err
	}

	go m.keepAliveLoop(channel, sessionDTO.Session.ID)
	go m.checkSessionIP(dialog, channel, consumerID, sessionDTO.Session.ID, originalPublicIP)

	return err
}

// checkSessionIP checks if IP has changed after connection was established.
func (m *connectionManager) checkSessionIP(dialog communication.Dialog, channel p2p.Channel, consumerID identity.Identity, sessionID session.ID, originalPublicIP string) {
	for i := 1; i <= m.config.IPCheck.MaxAttempts; i++ {
		// Skip check if not connected. This may happen when context was canceled via Disconnect.
		if m.Status().State != Connected {
			return
		}

		newPublicIP := m.getPublicIP()
		// If ip is changed notify peer that connection is successful.
		if originalPublicIP != newPublicIP {
			m.sendSessionStatus(dialog, channel, consumerID, sessionID, connectivity.StatusConnectionOk, nil)
			return
		}

		// Notify peer and quality oracle that ip is not changed after tunnel connection was established.
		if i == m.config.IPCheck.MaxAttempts {
			m.sendSessionStatus(dialog, channel, consumerID, sessionID, connectivity.StatusSessionIPNotChanged, nil)
			m.publishStateEvent(StateIPNotChanged)
			return
		}

		time.Sleep(m.config.IPCheck.SleepDurationAfterCheck)
	}
}

// sendSessionStatus sends session connectivity status to other peer.
func (m *connectionManager) sendSessionStatus(dialog communication.Dialog, channel p2p.ChannelSender, consumerID identity.Identity, sessionID session.ID, code connectivity.StatusCode, errDetails error) error {
	var errDetailsMsg string
	if errDetails != nil {
		errDetailsMsg = errDetails.Error()
	}

	if channel == nil {
		return m.connectivityStatusSender.Send(dialog, &connectivity.StatusMessage{
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

	ctx, cancel := context.WithTimeout(m.currentCtx(), 20*time.Second)
	defer cancel()
	_, err := channel.Send(ctx, p2p.TopicSessionStatus, p2p.ProtoMessage(sessionStatus))
	if err != nil {
		return fmt.Errorf("could not send p2p session status message: %w", err)
	}

	return nil
}

func (m *connectionManager) getPublicIP() string {
	currentPublicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		log.Error().Err(err).Msg("Could not get current public IP")
		return ""
	}
	return currentPublicIP
}

func (m *connectionManager) launchPayments(paymentInfo session.PaymentInfo, dialog communication.Dialog, channel p2p.Channel, consumerID, providerID identity.Identity, accountantID common.Address, proposal market.ServiceProposal, sessionID session.ID) error {
	payments, err := m.paymentEngineFactory(paymentInfo, dialog, channel, consumerID, providerID, accountantID, proposal, string(sessionID))
	if err != nil {
		return err
	}
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: payments")
		defer log.Trace().Msg("Cleaning: payments DONE")
		payments.Stop()
		return nil
	})

	go m.payForService(payments)
	return nil
}

func (m *connectionManager) cleanConnection() {
	m.cleanupLock.Lock()
	defer m.cleanupLock.Unlock()

	for i := len(m.cleanup) - 1; i >= 0; i-- {
		log.Trace().Msgf("Connection cleaning up: (%v/%v)", i+1, len(m.cleanup))
		err := m.cleanup[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	m.cleanup = nil
}

func (m *connectionManager) cleanAfterDisconnect() {
	m.cleanupLock.Lock()
	defer m.cleanupLock.Unlock()

	for i := len(m.cleanupAfterDisconnect) - 1; i >= 0; i-- {
		log.Trace().Msgf("Connection cleaning up (after disconnect): (%v/%v)", i+1, len(m.cleanupAfterDisconnect))
		err := m.cleanupAfterDisconnect[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	m.cleanupAfterDisconnect = nil
}

func (m *connectionManager) createDialog(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error) {
	dialog, err := m.newDialog(consumerID, providerID, contact)
	if err != nil {
		return nil, err
	}

	m.addCleanupAfterDisconnect(func() error {
		log.Trace().Msg("Cleaning: closing dialog")
		defer log.Trace().Msg("Cleaning: closing dialog DONE")
		return dialog.Close()
	})
	return dialog, err
}

func (m *connectionManager) createP2PChannel(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef p2p.ContactDefinition) (p2p.Channel, error) {
	channel, err := m.p2pDialer.Dial(ctx, consumerID, providerID, serviceType, contactDef)
	if err != nil {
		return nil, err
	}
	m.addCleanupAfterDisconnect(func() error {
		log.Trace().Msg("Cleaning: closing P2P communication channel")
		defer log.Trace().Msg("Cleaning: P2P communication channel DONE")

		return channel.Close()
	})
	return channel, nil
}

func (m *connectionManager) addCleanupAfterDisconnect(fn func() error) {
	m.cleanupLock.Lock()
	defer m.cleanupLock.Unlock()
	m.cleanupAfterDisconnect = append(m.cleanupAfterDisconnect, fn)
}

func (m *connectionManager) addCleanup(fn func() error) {
	m.cleanupLock.Lock()
	defer m.cleanupLock.Unlock()
	m.cleanup = append(m.cleanup, fn)
}

func (m *connectionManager) createP2PSession(ctx context.Context, c Connection, p2pChannel p2p.ChannelSender, consumerID identity.Identity, accountantID common.Address, proposal market.ServiceProposal) (session.CreateResponse, error) {
	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return session.CreateResponse{}, fmt.Errorf("could not get session config: %w", err)
	}

	config, err := json.Marshal(sessionCreateConfig)
	if err != nil {
		return session.CreateResponse{}, fmt.Errorf("could not marshal session config: %w", err)
	}

	sessionRequest := &pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:             consumerID.Address,
			AccountantID:   accountantID.Hex(),
			PaymentVersion: string(session.PaymentVersionV3),
		},
		ProposalID: int64(proposal.ID),
		Config:     config,
	}
	log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionCreate, sessionRequest.String())
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	res, err := p2pChannel.Send(ctx, p2p.TopicSessionCreate, p2p.ProtoMessage(sessionRequest))
	if err != nil {
		return session.CreateResponse{}, fmt.Errorf("could not send p2p session create request: %w", err)
	}

	var sessionResponse pb.SessionResponse
	err = res.UnmarshalProto(&sessionResponse)
	if err != nil {
		return session.CreateResponse{}, fmt.Errorf("could not unmarshal session reply to proto: %w", err)
	}

	m.acknowledge = func() {
		pc := &pb.SessionInfo{
			ConsumerID: consumerID.Address,
			SessionID:  sessionResponse.GetID(),
		}
		log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionAcknowledge, pc.String())
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := p2pChannel.Send(ctx, p2p.TopicSessionAcknowledge, p2p.ProtoMessage(pc))
		if err != nil {
			log.Warn().Err(err).Msg("Acknowledge failed")
		}
	}
	m.addCleanupAfterDisconnect(func() error {
		log.Trace().Msg("Cleaning: requesting session destroy")
		defer log.Trace().Msg("Cleaning: requesting session destroy DONE")

		sessionDestroy := &pb.SessionInfo{
			ConsumerID: consumerID.Address,
			SessionID:  sessionResponse.GetID(),
		}

		log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionDestroy, sessionDestroy.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := p2pChannel.Send(ctx, p2p.TopicSessionDestroy, p2p.ProtoMessage(sessionDestroy))
		if err != nil {
			return fmt.Errorf("could not send session destroy request: %w", err)
		}

		return nil
	})

	sessionID := session.ID(sessionResponse.GetID())
	m.publishSessionCreate(sessionID)

	return session.CreateResponse{
		Session: session.SessionDto{
			ID:     sessionID,
			Config: sessionResponse.GetConfig(),
		},
		PaymentInfo: session.PaymentInfo{Supports: sessionResponse.GetPaymentInfo()},
	}, nil
}

func (m *connectionManager) createSession(c Connection, dialog communication.Dialog, consumerID identity.Identity, accountantID common.Address, proposal market.ServiceProposal) (session.CreateResponse, error) {
	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return session.CreateResponse{}, fmt.Errorf("could not get session config: %w", err)
	}

	config, err := json.Marshal(sessionCreateConfig)
	if err != nil {
		return session.CreateResponse{}, fmt.Errorf("could not marshal session config: %w", err)
	}

	sessionResponse, err := session.RequestSessionCreate(dialog, session.CreateRequest{
		ProposalID: proposal.ID,
		Config:     config,
		ConsumerInfo: &session.ConsumerInfo{
			IssuerID:       consumerID,
			AccountantID:   identity.FromAddress(accountantID.Hex()),
			PaymentVersion: session.PaymentVersionV3,
		},
	})
	if err != nil {
		return session.CreateResponse{}, err
	}

	m.acknowledge = func() {
		err := session.AcknowledgeSession(dialog, string(sessionResponse.Session.ID))
		if err != nil {
			log.Warn().Err(err).Msg("Acknowledge failed")
		}
	}
	m.addCleanupAfterDisconnect(func() error {
		log.Trace().Msg("Cleaning: requesting session destroy")
		defer log.Trace().Msg("Cleaning: requesting session destroy DONE")
		return session.RequestSessionDestroy(dialog, sessionResponse.Session.ID)
	})

	m.publishSessionCreate(sessionResponse.Session.ID)

	return sessionResponse, nil
}

func (m *connectionManager) publishSessionCreate(sessionID session.ID) {
	m.setStatus(func(status *Status) {
		status.SessionID = sessionID
	})

	m.eventPublisher.Publish(AppTopicConnectionSession, AppEventConnectionSession{
		Status:      SessionCreatedStatus,
		SessionInfo: m.Status(),
	})

	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: publishing session ended status")
		defer log.Trace().Msg("Cleaning: publishing session ended status DONE")
		m.eventPublisher.Publish(AppTopicConnectionSession, AppEventConnectionSession{
			Status:      SessionEndedStatus,
			SessionInfo: m.Status(),
		})
		return nil
	})
}

func (m *connectionManager) startConnection(ctx context.Context, conn Connection, disableKillSwitch bool, connectOptions ConnectOptions) (err error) {
	if err = conn.Start(ctx, connectOptions); err != nil {
		return err
	}
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: stopping connection")
		defer log.Trace().Msg("Cleaning: stopping connection DONE")
		conn.Stop()
		return nil
	})

	statsPublisher := newStatsPublisher(m.eventPublisher, m.statsReportInterval)
	go statsPublisher.start(m, conn)
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: stopping statistics publisher")
		defer log.Trace().Msg("Cleaning: stopping statistics publisher DONE")
		statsPublisher.stop()
		return nil
	})

	err = m.setupTrafficBlock(disableKillSwitch)
	if err != nil {
		return err
	}

	err = m.waitForConnectedState(conn.State())
	if err != nil {
		return err
	}

	go m.consumeConnectionStates(conn.State())
	go m.connectionWaiter(conn)
	return nil
}

func (m *connectionManager) Status() Status {
	m.statusLock.RLock()
	defer m.statusLock.RUnlock()

	return m.status
}

func (m *connectionManager) setStatus(delta func(status *Status)) {
	m.statusLock.Lock()
	stateWas := m.status.State

	delta(&m.status)

	state := m.status.State
	m.statusLock.Unlock()

	if state != stateWas {
		log.Info().Msgf("Connection state: %v -> %v", stateWas, state)
		m.publishStateEvent(m.status.State)
	}
}

func (m *connectionManager) statusConnecting(consumerID identity.Identity, accountantID common.Address, proposal market.ServiceProposal) {
	m.setStatus(func(status *Status) {
		*status = Status{
			StartedAt:    m.timeGetter(),
			ConsumerID:   consumerID,
			AccountantID: accountantID,
			Proposal:     proposal,
			State:        Connecting,
		}
	})
}

func (m *connectionManager) statusConnected() {
	m.setStatus(func(status *Status) {
		status.State = Connected
	})
}

func (m *connectionManager) statusReconnecting() {
	m.setStatus(func(status *Status) {
		status.State = Reconnecting
	})
}

func (m *connectionManager) statusNotConnected() {
	m.setStatus(func(status *Status) {
		status.State = NotConnected
	})
}

func (m *connectionManager) statusDisconnecting() {
	m.setStatus(func(status *Status) {
		status.State = Disconnecting
	})
}

func (m *connectionManager) statusCanceled() {
	m.setStatus(func(status *Status) {
		status.State = Canceled
	})
}

func (m *connectionManager) Cancel() {
	m.statusCanceled()
	logDisconnectError(m.Disconnect())
}

func (m *connectionManager) Disconnect() error {
	if m.Status().State == NotConnected {
		return ErrNoConnection
	}

	m.statusDisconnecting()
	m.disconnect()

	return nil
}

func (m *connectionManager) disconnect() {
	m.discoLock.Lock()
	defer m.discoLock.Unlock()

	m.ctxLock.Lock()
	m.cancel()
	m.ctxLock.Unlock()

	m.cleanConnection()
	m.statusNotConnected()

	m.cleanAfterDisconnect()
}

func (m *connectionManager) payForService(payments PaymentIssuer) {
	err := payments.Start()
	if err != nil {
		log.Error().Err(err).Msg("Payment error")
		err = m.Disconnect()
		if err != nil {
			log.Error().Err(err).Msg("Could not disconnect gracefully")
		}
	}
}

func (m *connectionManager) connectionWaiter(connection Connection) {
	err := connection.Wait()
	if err != nil {
		log.Warn().Err(err).Msg("Connection exited with error")
	} else {
		log.Info().Msg("Connection exited")
	}

	logDisconnectError(m.Disconnect())
}

func (m *connectionManager) waitForConnectedState(stateChannel <-chan State) error {
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
				if m.acknowledge != nil {
					go m.acknowledge()
				}
				m.onStateChanged(state)
				return nil
			default:
				m.onStateChanged(state)
			}
		case <-m.currentCtx().Done():
			return m.currentCtx().Err()
		}
	}
}

func (m *connectionManager) consumeConnectionStates(stateChannel <-chan State) {
	for state := range stateChannel {
		m.onStateChanged(state)
	}

	log.Debug().Msg("State updater stopCalled")
	logDisconnectError(m.Disconnect())
}

func (m *connectionManager) onStateChanged(state State) {
	log.Debug().Msgf("Connection state received: %s", state)
	m.publishStateEvent(state)

	// React just to certains stains from connection. Because disconnect happens in connectionWaiter
	switch state {
	case Connected:
		m.statusConnected()
	case Reconnecting:
		m.statusReconnecting()
	}
}

func (m *connectionManager) setupTrafficBlock(disableKillSwitch bool) error {
	if disableKillSwitch {
		return nil
	}

	outboundIP, err := m.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return err
	}

	removeRule, err := firewall.BlockNonTunnelTraffic(firewall.Session, outboundIP)
	if err != nil {
		return err
	}
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: traffic block rule")
		defer log.Trace().Msg("Cleaning: traffic block rule DONE")
		removeRule()
		return nil
	})
	return nil
}

func (m *connectionManager) publishStateEvent(state State) {
	m.eventPublisher.Publish(AppTopicConnectionState, AppEventConnectionState{
		State:       state,
		SessionInfo: m.Status(),
	})
}

func (m *connectionManager) keepAliveLoop(channel p2p.Channel, sessionID session.ID) {
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
		case <-m.currentCtx().Done():
			log.Debug().Msgf("Stopping p2p keepalive: %v", m.currentCtx().Err())
			return
		case <-time.After(m.config.KeepAlive.SendInterval):
			if err := m.sendKeepAlivePing(channel, sessionID); err != nil {
				log.Err(err).Msgf("Failed to send p2p keepalive ping. SessionID=%s", sessionID)
				errCount++
				if errCount == m.config.KeepAlive.MaxSendErrCount {
					log.Error().Msgf("Max p2p keepalive err count reached, disconnecting. SessionID=%s", sessionID)
					m.Disconnect()
					return
				}
			} else {
				errCount = 0
			}
		}
	}
}

func (m *connectionManager) sendKeepAlivePing(channel p2p.Channel, sessionID session.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.KeepAlive.SendTimeout)
	defer cancel()
	msg := &pb.P2PKeepAlivePing{
		SessionID: string(sessionID),
	}
	_, err := channel.Send(ctx, p2p.TopicKeepAlive, p2p.ProtoMessage(msg))
	return err
}

func (m *connectionManager) currentCtx() context.Context {
	m.ctxLock.RLock()
	defer m.ctxLock.RUnlock()

	return m.ctx
}

func logDisconnectError(err error) {
	if err != nil && err != ErrNoConnection {
		log.Error().Err(err).Msg("Disconnect error")
	}
}
