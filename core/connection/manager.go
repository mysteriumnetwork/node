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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/trace"
)

const (
	p2pDialTimeout = 60 * time.Second
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
			SendInterval:    5 * time.Second,
			SendTimeout:     5 * time.Second,
			MaxSendErrCount: 3,
		},
	}
}

// Creator creates new connection by given options and uses state channel to report state changes
type Creator func(serviceType string) (Connection, error)

// ConnectionStart start new connection with a given options.
type ConnectionStart func(context.Context, ConnectOptions) error

// PaymentIssuer handles the payments for service
type PaymentIssuer interface {
	Start() error
	SetSessionID(string)
	Stop()
}

// PriceGetter fetches the current price.
type PriceGetter interface {
	GetCurrentPrice(nodeType string, country string) (market.Price, error)
}

type validator interface {
	Validate(chainID int64, consumerID identity.Identity, p market.Price) error
}

// TimeGetter function returns current time
type TimeGetter func() time.Time

// PaymentEngineFactory creates a new payment issuer from the given params
type PaymentEngineFactory func(senderUUID string, channel p2p.Channel, consumer, provider identity.Identity, hermes common.Address, proposal proposal.PricedServiceProposal, price market.Price) (PaymentIssuer, error)

// ProposalLookup returns a service proposal based on predefined conditions.
type ProposalLookup func() (proposal *proposal.PricedServiceProposal, err error)

type connectionManager struct {
	// These are passed on creation.
	paymentEngineFactory PaymentEngineFactory
	newConnection        Creator
	eventBus             eventbus.EventBus
	ipResolver           ip.Resolver
	locationResolver     location.OriginResolver
	config               Config
	statsReportInterval  time.Duration
	validator            validator
	p2pDialer            p2p.Dialer
	timeGetter           TimeGetter

	// These are populated by Connect at runtime.
	ctx                    context.Context
	ctxLock                sync.RWMutex
	status                 connectionstate.Status
	statusLock             sync.RWMutex
	cleanupLock            sync.Mutex
	cleanup                []func() error
	cleanupAfterDisconnect []func() error
	cleanupFinished        chan struct{}
	cleanupFinishedLock    sync.Mutex
	acknowledge            func()
	cancel                 func()
	channel                p2p.Channel

	preReconnect  func()
	postReconnect func()

	discoLock      sync.Mutex
	connectOptions ConnectOptions

	activeConnection Connection
	statsTracker     statsTracker

	uuid string

	provChecker *ProviderChecker
}

// NewManager creates connection manager with given dependencies
func NewManager(
	paymentEngineFactory PaymentEngineFactory,
	connectionCreator Creator,
	eventBus eventbus.EventBus,
	ipResolver ip.Resolver,
	locationResolver location.OriginResolver,
	config Config,
	statsReportInterval time.Duration,
	validator validator,
	p2pDialer p2p.Dialer,
	preReconnect, postReconnect func(),
	provChecker *ProviderChecker,
) *connectionManager {
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err) // This should never happen.
	}

	m := &connectionManager{
		newConnection:        connectionCreator,
		status:               connectionstate.Status{State: connectionstate.NotConnected},
		eventBus:             eventBus,
		paymentEngineFactory: paymentEngineFactory,
		cleanup:              make([]func() error, 0),
		cleanupFinished:      make(chan struct{}, 1),
		ipResolver:           ipResolver,
		locationResolver:     locationResolver,
		config:               config,
		statsReportInterval:  statsReportInterval,
		validator:            validator,
		p2pDialer:            p2pDialer,
		timeGetter:           time.Now,
		preReconnect:         preReconnect,
		postReconnect:        postReconnect,
		uuid:                 uuid.String(),
		provChecker:          provChecker,
	}

	m.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, m.reconnectOnHold)

	return m
}

func (m *connectionManager) chainID() int64 {
	return config.GetInt64(config.FlagChainID)
}

func (m *connectionManager) Connect(consumerID identity.Identity, hermesID common.Address, proposalLookup ProposalLookup, params ConnectParams) (err error) {
	var sessionID session.ID

	proposal, err := proposalLookup()
	if err != nil {
		return fmt.Errorf("failed to lookup proposal: %w", err)
	}

	tracer := trace.NewTracer("Consumer whole Connect")
	defer func() {
		traceResult := tracer.Finish(m.eventBus, string(sessionID))
		log.Debug().Msgf("Consumer connection trace: %s", traceResult)
	}()

	// make sure cache is cleared when connect terminates at any stage as part of disconnect
	// we assume that IPResolver might be used / cache IP before connect
	m.addCleanup(func() error {
		m.clearIPCache()
		return nil
	})

	if m.Status().State != connectionstate.NotConnected {
		return ErrAlreadyExists
	}

	prc := m.priceFromProposal(*proposal)

	err = m.validator.Validate(m.chainID(), consumerID, prc)
	if err != nil {
		return err
	}

	m.ctxLock.Lock()
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.ctxLock.Unlock()

	m.statusConnecting(consumerID, hermesID, *proposal)
	defer func() {
		if err != nil {
			log.Err(err).Msg("Connect failed, disconnecting")
			m.disconnect()
		}
	}()

	m.connectOptions = ConnectOptions{
		ConsumerID:     consumerID,
		HermesID:       hermesID,
		Proposal:       *proposal,
		ProposalLookup: proposalLookup,
		Params:         params,
	}

	m.activeConnection, err = m.newConnection(proposal.ServiceType)
	if err != nil {
		return err
	}

	sessionID, err = m.initSession(tracer, prc)
	if err != nil {
		return err
	}

	originalPublicIP := m.getPublicIP()

	err = m.startConnection(m.currentCtx(), m.activeConnection, m.activeConnection.Start, m.connectOptions, tracer)
	if err != nil {
		return m.handleStartError(sessionID, err)
	}

	err = m.waitForConnectedState(m.activeConnection.State())
	if err != nil {
		return m.handleStartError(sessionID, err)
	}

	m.statsTracker = newStatsTracker(m.eventBus, m.statsReportInterval)
	go m.statsTracker.start(m, m.activeConnection)
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: stopping statistics publisher")
		defer log.Trace().Msg("Cleaning: stopping statistics publisher DONE")
		m.statsTracker.stop()
		return nil
	})

	if m.provChecker != nil {
		go m.provChecker.Diag(m, proposal.ProviderID)
	}

	go m.consumeConnectionStates(m.activeConnection.State())
	go m.checkSessionIP(m.channel, m.connectOptions.ConsumerID, m.connectOptions.SessionID, originalPublicIP)

	return nil
}

func (m *connectionManager) autoReconnect() (err error) {
	var sessionID session.ID

	tracer := trace.NewTracer("Consumer whole autoReconnect")
	defer func() {
		traceResult := tracer.Finish(m.eventBus, string(sessionID))
		log.Debug().Msgf("Consumer connection trace: %s", traceResult)
	}()

	proposal, err := m.connectOptions.ProposalLookup()
	if err != nil {
		return fmt.Errorf("failed to lookup proposal: %w", err)
	}

	m.connectOptions.Proposal = *proposal

	sessionID, err = m.initSession(tracer, m.priceFromProposal(m.connectOptions.Proposal))
	if err != nil {
		return err
	}

	err = m.startConnection(m.currentCtx(), m.activeConnection, m.activeConnection.Reconnect, m.connectOptions, tracer)
	if err != nil {
		return m.handleStartError(sessionID, err)
	}

	return nil
}

func (m *connectionManager) priceFromProposal(proposal proposal.PricedServiceProposal) market.Price {
	p := market.Price{
		PricePerHour: proposal.Price.PricePerHour,
		PricePerGiB:  proposal.Price.PricePerGiB,
	}

	if config.GetBool(config.FlagPaymentsDuringSessionDebug) {
		log.Info().Msg("Payments debug bas been enabled, will use absurd amounts for the proposal price")
		amount := config.GetUInt64(config.FlagPaymentsAmountDuringSessionDebug)
		if amount == 0 {
			amount = 5000000000000000000
		}

		p = market.Price{
			PricePerHour: new(big.Int).SetUint64(amount),
			PricePerGiB:  new(big.Int).SetUint64(amount),
		}
	}

	return p
}

func (m *connectionManager) initSession(tracer *trace.Tracer, prc market.Price) (sessionID session.ID, err error) {
	err = m.createP2PChannel(m.connectOptions, tracer)
	if err != nil {
		return sessionID, fmt.Errorf("could not create p2p channel during connect: %w", err)
	}

	m.connectOptions.ProviderNATConn = m.channel.ServiceConn()
	m.connectOptions.ChannelConn = m.channel.Conn()

	paymentSession, err := m.paymentLoop(m.connectOptions, prc)
	if err != nil {
		return sessionID, err
	}

	sessionDTO, err := m.createP2PSession(m.activeConnection, m.connectOptions, tracer, prc)
	sessionID = session.ID(sessionDTO.GetID())
	if err != nil {
		m.sendSessionStatus(m.channel, m.connectOptions.ConsumerID, sessionID, connectivity.StatusSessionEstablishmentFailed, err)
		return sessionID, err
	}

	traceStart := tracer.StartStage("Consumer session creation (start)")
	go m.keepAliveLoop(m.channel, sessionID)
	m.setStatus(func(status *connectionstate.Status) {
		status.SessionID = sessionID
	})
	m.publishSessionCreate(sessionID)
	paymentSession.SetSessionID(string(sessionID))
	tracer.EndStage(traceStart)

	m.connectOptions.SessionID = sessionID
	m.connectOptions.SessionConfig = sessionDTO.GetConfig()

	return sessionID, nil
}

func (m *connectionManager) handleStartError(sessionID session.ID, err error) error {
	if errors.Is(err, context.Canceled) {
		return ErrConnectionCancelled
	}
	m.addCleanupAfterDisconnect(func() error {
		return m.sendSessionStatus(m.channel, m.connectOptions.ConsumerID, sessionID, connectivity.StatusConnectionFailed, err)
	})
	m.publishStateEvent(connectionstate.StateConnectionFailed)

	log.Info().Err(err).Msg("Cancelling connection initiation: ")
	m.Cancel()
	return err
}

func (m *connectionManager) clearIPCache() {
	if config.GetBool(config.FlagProxyMode) || config.GetBool(config.FlagDVPNMode) {
		return
	}

	if cr, ok := m.ipResolver.(*ip.CachedResolver); ok {
		cr.ClearCache()
	}
}

// checkSessionIP checks if IP has changed after connection was established.
func (m *connectionManager) checkSessionIP(channel p2p.Channel, consumerID identity.Identity, sessionID session.ID, originalPublicIP string) {
	if config.GetBool(config.FlagProxyMode) || config.GetBool(config.FlagDVPNMode) {
		return
	}

	for i := 1; i <= m.config.IPCheck.MaxAttempts; i++ {
		// Skip check if not connected. This may happen when context was canceled via Disconnect.
		if m.Status().State != connectionstate.Connected {
			return
		}

		newPublicIP := m.getPublicIP()
		// If ip is changed notify peer that connection is successful.
		if originalPublicIP != newPublicIP {
			m.sendSessionStatus(channel, consumerID, sessionID, connectivity.StatusConnectionOk, nil)
			return
		}

		// Notify peer and quality oracle that ip is not changed after tunnel connection was established.
		if i == m.config.IPCheck.MaxAttempts {
			m.sendSessionStatus(channel, consumerID, sessionID, connectivity.StatusSessionIPNotChanged, nil)
			m.publishStateEvent(connectionstate.StateIPNotChanged)
			return
		}

		time.Sleep(m.config.IPCheck.SleepDurationAfterCheck)
	}
}

// sendSessionStatus sends session connectivity status to other peer.
func (m *connectionManager) sendSessionStatus(channel p2p.ChannelSender, consumerID identity.Identity, sessionID session.ID, code connectivity.StatusCode, errDetails error) error {
	var errDetailsMsg string
	if errDetails != nil {
		errDetailsMsg = errDetails.Error()
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

func (m *connectionManager) paymentLoop(opts ConnectOptions, price market.Price) (PaymentIssuer, error) {
	payments, err := m.paymentEngineFactory(m.uuid, m.channel, opts.ConsumerID, identity.FromAddress(opts.Proposal.ProviderID), opts.HermesID, opts.Proposal, price)
	if err != nil {
		return nil, err
	}
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: payments")
		defer log.Trace().Msg("Cleaning: payments DONE")
		payments.Stop()
		return nil
	})

	go func() {
		err := payments.Start()
		if err != nil {
			log.Error().Err(err).Msg("Payment error")

			if config.GetBool(config.FlagKeepConnectedOnFail) {
				m.statusOnHold()
			} else {
				err = m.Disconnect()
				if err != nil {
					log.Error().Err(err).Msg("Could not disconnect gracefully")
				}
			}
		}
	}()
	return payments, nil
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

func (m *connectionManager) createP2PChannel(opts ConnectOptions, tracer *trace.Tracer) error {
	trace := tracer.StartStage("Consumer P2P channel creation")
	defer tracer.EndStage(trace)

	contactDef, err := p2p.ParseContact(opts.Proposal.Contacts)
	if err != nil {
		return fmt.Errorf("provider does not support p2p communication: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(m.currentCtx(), p2pDialTimeout)
	defer cancel()

	// TODO register all handlers before channel read/write loops
	channel, err := m.p2pDialer.Dial(timeoutCtx, opts.ConsumerID, identity.FromAddress(opts.Proposal.ProviderID), opts.Proposal.ServiceType, contactDef, tracer)
	if err != nil {
		return fmt.Errorf("p2p dialer failed: %w", err)
	}
	m.addCleanupAfterDisconnect(func() error {
		log.Trace().Msg("Cleaning: closing P2P communication channel")
		defer log.Trace().Msg("Cleaning: P2P communication channel DONE")

		return channel.Close()
	})

	m.channel = channel
	return nil
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

func (m *connectionManager) createP2PSession(c Connection, opts ConnectOptions, tracer *trace.Tracer, requestedPrice market.Price) (*pb.SessionResponse, error) {
	trace := tracer.StartStage("Consumer session creation")
	defer tracer.EndStage(trace)

	sessionCreateConfig, err := c.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get session config: %w", err)
	}

	config, err := json.Marshal(sessionCreateConfig)
	if err != nil {
		return nil, fmt.Errorf("could not marshal session config: %w", err)
	}

	sessionRequest := &pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:             opts.ConsumerID.Address,
			HermesID:       opts.HermesID.Hex(),
			PaymentVersion: "v3",
			Location: &pb.LocationInfo{
				Country: m.Status().ConsumerLocation.Country,
			},
			Pricing: &pb.Pricing{
				PerGib:  requestedPrice.PricePerGiB.Bytes(),
				PerHour: requestedPrice.PricePerHour.Bytes(),
			},
		},
		ProposalID: opts.Proposal.ID,
		Config:     config,
	}
	log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionCreate, sessionRequest.String())
	ctx, cancel := context.WithTimeout(m.currentCtx(), 20*time.Second)
	defer cancel()
	res, err := m.channel.Send(ctx, p2p.TopicSessionCreate, p2p.ProtoMessage(sessionRequest))
	if err != nil {
		return nil, fmt.Errorf("could not send p2p session create request: %w", err)
	}

	var sessionResponse pb.SessionResponse
	err = res.UnmarshalProto(&sessionResponse)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal session reply to proto: %w", err)
	}

	channel := m.channel
	m.acknowledge = func() {
		pc := &pb.SessionInfo{
			ConsumerID: opts.ConsumerID.Address,
			SessionID:  sessionResponse.GetID(),
		}
		log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionAcknowledge, pc.String())
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := channel.Send(ctx, p2p.TopicSessionAcknowledge, p2p.ProtoMessage(pc))
		if err != nil {
			log.Warn().Err(err).Msg("Acknowledge failed")
		}
	}
	m.addCleanupAfterDisconnect(func() error {
		log.Trace().Msg("Cleaning: requesting session destroy")
		defer log.Trace().Msg("Cleaning: requesting session destroy DONE")

		sessionDestroy := &pb.SessionInfo{
			ConsumerID: opts.ConsumerID.Address,
			SessionID:  sessionResponse.GetID(),
		}

		log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionDestroy, sessionDestroy.String())
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := m.channel.Send(ctx, p2p.TopicSessionDestroy, p2p.ProtoMessage(sessionDestroy))
		if err != nil {
			return fmt.Errorf("could not send session destroy request: %w", err)
		}

		return nil
	})

	return &sessionResponse, nil
}

func (m *connectionManager) publishSessionCreate(sessionID session.ID) {
	sessionInfo := m.Status()
	// avoid printing IP address in logs
	sessionInfo.ConsumerLocation.IP = ""

	m.eventBus.Publish(connectionstate.AppTopicConnectionSession, connectionstate.AppEventConnectionSession{
		Status:      connectionstate.SessionCreatedStatus,
		SessionInfo: sessionInfo,
	})

	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: publishing session ended status")
		defer log.Trace().Msg("Cleaning: publishing session ended status DONE")

		sessionInfo := m.Status()
		// avoid printing IP address in logs
		sessionInfo.ConsumerLocation.IP = ""

		m.eventBus.Publish(connectionstate.AppTopicConnectionSession, connectionstate.AppEventConnectionSession{
			Status:      connectionstate.SessionEndedStatus,
			SessionInfo: sessionInfo,
		})
		return nil
	})
}

func (m *connectionManager) startConnection(ctx context.Context, conn Connection, start ConnectionStart, connectOptions ConnectOptions, tracer *trace.Tracer) (err error) {
	trace := tracer.StartStage("Consumer start connection")
	defer tracer.EndStage(trace)

	if err = start(ctx, connectOptions); err != nil {
		return err
	}
	m.addCleanup(func() error {
		log.Trace().Msg("Cleaning: stopping connection")
		defer log.Trace().Msg("Cleaning: stopping connection DONE")
		conn.Stop()
		return nil
	})

	err = m.setupTrafficBlock(connectOptions.Params.DisableKillSwitch)
	if err != nil {
		return err
	}

	// Clear IP cache so session IP check can report that IP has really changed.
	m.clearIPCache()

	return nil
}

func (m *connectionManager) Status() connectionstate.Status {
	m.statusLock.RLock()
	defer m.statusLock.RUnlock()

	return m.status
}

func (m *connectionManager) UUID() string {
	m.statusLock.RLock()
	defer m.statusLock.RUnlock()

	return m.uuid
}

func (m *connectionManager) Stats() connectionstate.Statistics {
	return m.statsTracker.stats()
}

func (m *connectionManager) setStatus(delta func(status *connectionstate.Status)) {
	m.statusLock.Lock()
	stateWas := m.status.State

	delta(&m.status)

	state := m.status.State
	m.statusLock.Unlock()

	if state != stateWas {
		log.Info().Msgf("Connection state: %v -> %v", stateWas, state)
		m.publishStateEvent(state)
	}
}

func (m *connectionManager) statusConnecting(consumerID identity.Identity, accountantID common.Address, proposal proposal.PricedServiceProposal) {
	m.setStatus(func(status *connectionstate.Status) {
		*status = connectionstate.Status{
			StartedAt:        m.timeGetter(),
			ConsumerID:       consumerID,
			ConsumerLocation: m.locationResolver.GetOrigin(),
			HermesID:         accountantID,
			Proposal:         proposal,
			State:            connectionstate.Connecting,
		}
	})
}

func (m *connectionManager) statusConnected() {
	m.setStatus(func(status *connectionstate.Status) {
		status.State = connectionstate.Connected
	})
}

func (m *connectionManager) statusReconnecting() {
	m.setStatus(func(status *connectionstate.Status) {
		status.State = connectionstate.Reconnecting
	})
}

func (m *connectionManager) statusNotConnected() {
	m.setStatus(func(status *connectionstate.Status) {
		status.State = connectionstate.NotConnected
	})
}

func (m *connectionManager) statusDisconnecting() {
	m.setStatus(func(status *connectionstate.Status) {
		status.State = connectionstate.Disconnecting
	})
}

func (m *connectionManager) statusCanceled() {
	m.setStatus(func(status *connectionstate.Status) {
		status.State = connectionstate.Canceled
	})
}

func (m *connectionManager) statusOnHold() {
	m.setStatus(func(status *connectionstate.Status) {
		status.State = connectionstate.StateOnHold
	})
}

func (m *connectionManager) Cancel() {
	m.statusCanceled()
	logDisconnectError(m.Disconnect())
}

func (m *connectionManager) Disconnect() error {
	if m.Status().State == connectionstate.NotConnected {
		return ErrNoConnection
	}

	m.statusDisconnecting()
	m.disconnect()

	return nil
}

func (m *connectionManager) CheckChannel(ctx context.Context) error {
	if err := m.sendKeepAlivePing(ctx, m.channel, m.Status().SessionID); err != nil {
		return fmt.Errorf("keep alive ping failed: %w", err)
	}
	return nil
}

func (m *connectionManager) disconnect() {
	m.discoLock.Lock()
	defer m.discoLock.Unlock()

	m.cleanupFinishedLock.Lock()
	defer m.cleanupFinishedLock.Unlock()
	m.cleanupFinished = make(chan struct{})
	defer close(m.cleanupFinished)

	m.ctxLock.Lock()
	m.cancel()
	m.ctxLock.Unlock()

	m.cleanConnection()
	m.statusNotConnected()

	m.cleanAfterDisconnect()
}

func (m *connectionManager) waitForConnectedState(stateChannel <-chan connectionstate.State) error {
	log.Debug().Msg("waiting for connected state")
	for {
		select {
		case state, more := <-stateChannel:
			if !more {
				return ErrConnectionFailed
			}

			switch state {
			case connectionstate.Connected:
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

func (m *connectionManager) consumeConnectionStates(stateChannel <-chan connectionstate.State) {
	for state := range stateChannel {
		m.onStateChanged(state)
	}
}

func (m *connectionManager) onStateChanged(state connectionstate.State) {
	log.Debug().Msgf("Connection state received: %s", state)

	// React just to certain stains from connection. Because disconnect happens in connectionWaiter
	switch state {
	case connectionstate.Connected:
		m.statusConnected()
	case connectionstate.Reconnecting:
		m.statusReconnecting()
	}
}

func (m *connectionManager) setupTrafficBlock(disableKillSwitch bool) error {
	if disableKillSwitch {
		return nil
	}

	outboundIP, err := m.ipResolver.GetOutboundIP()
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

func (m *connectionManager) reconnectOnHold(state connectionstate.AppEventConnectionState) {
	if state.State != connectionstate.StateOnHold || !config.GetBool(config.FlagAutoReconnect) {
		return
	}

	if m.channel != nil {
		m.channel.Close()
	}

	m.preReconnect()
	m.clearIPCache()

	for err := m.autoReconnect(); err != nil; err = m.autoReconnect() {
		select {
		case <-m.currentCtx().Done():
			log.Info().Err(m.currentCtx().Err()).Msg("Stopping reconnect")
			return
		default:
			log.Error().Err(err).Msg("Failed to reconnect active session, will try again")
		}
	}
	m.postReconnect()
}

func (m *connectionManager) publishStateEvent(state connectionstate.State) {
	sessionInfo := m.Status()
	// avoid printing IP address in logs
	sessionInfo.ConsumerLocation.IP = ""

	m.eventBus.Publish(connectionstate.AppTopicConnectionState, connectionstate.AppEventConnectionState{
		UUID:        m.uuid,
		State:       state,
		SessionInfo: sessionInfo,
	})
}

func (m *connectionManager) keepAliveLoop(channel p2p.Channel, sessionID session.ID) {
	// Register handler for handling p2p keep alive pings from provider.
	channel.Handle(p2p.TopicKeepAlive, func(c p2p.Context) error {
		var ping pb.P2PKeepAlivePing
		if err := c.Request().UnmarshalProto(&ping); err != nil {
			return err
		}

		log.Debug().Msgf("Received p2p keepalive ping with SessionID=%s from %s", ping.SessionID, c.PeerID().ToCommonAddress())
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
			ctx, cancel := context.WithTimeout(context.Background(), m.config.KeepAlive.SendTimeout)
			if err := m.sendKeepAlivePing(ctx, channel, sessionID); err != nil {
				log.Err(err).Msgf("Failed to send p2p keepalive ping. SessionID=%s", sessionID)
				errCount++
				if errCount == m.config.KeepAlive.MaxSendErrCount {
					log.Error().Msgf("Max p2p keepalive err count reached, disconnecting. SessionID=%s", sessionID)
					if config.GetBool(config.FlagKeepConnectedOnFail) {
						m.statusOnHold()
					} else {
						m.Disconnect()
					}
					cancel()
					return
				}
			} else {
				errCount = 0
			}
			cancel()
		}
	}
}

func (m *connectionManager) sendKeepAlivePing(ctx context.Context, channel p2p.Channel, sessionID session.ID) error {
	msg := &pb.P2PKeepAlivePing{
		SessionID: string(sessionID),
	}

	start := time.Now()
	_, err := channel.Send(ctx, p2p.TopicKeepAlive, p2p.ProtoMessage(msg))
	if err != nil {
		return err
	}

	m.eventBus.Publish(quality.AppTopicConsumerPingP2P, quality.PingEvent{
		SessionID: string(sessionID),
		Duration:  time.Since(start),
	})

	return nil
}

func (m *connectionManager) currentCtx() context.Context {
	m.ctxLock.RLock()
	defer m.ctxLock.RUnlock()

	return m.ctx
}

func (m *connectionManager) Reconnect() {
	err := m.Disconnect()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to disconnect stale session")
	}
	log.Info().Msg("Waiting for previous session to cleanup")

	m.cleanupFinishedLock.Lock()
	defer m.cleanupFinishedLock.Unlock()
	<-m.cleanupFinished
	err = m.Connect(m.connectOptions.ConsumerID, m.connectOptions.HermesID, m.connectOptions.ProposalLookup, m.connectOptions.Params)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to reconnect")
	}
}

func logDisconnectError(err error) {
	if err != nil && err != ErrNoConnection {
		log.Error().Err(err).Msg("Disconnect error")
	}
}
