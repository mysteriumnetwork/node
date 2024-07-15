/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
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
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

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

type conn struct {
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
	// statsTracker     statsTracker

	resChannel chan interface{}

	uuid string
}

type diagConnectionManager struct {
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

	// populated by Connect at runtime.
	connsMu sync.Mutex
	conns   map[string]*conn

	ratelimiter *rate.Limiter
}

// NewDiagManager creates connection manager with given dependencies
func NewDiagManager(
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
) *diagConnectionManager {

	m := &diagConnectionManager{
		conns: make(map[string]*conn),

		newConnection:        connectionCreator,
		eventBus:             eventBus,
		paymentEngineFactory: paymentEngineFactory,
		ipResolver:           ipResolver,
		locationResolver:     locationResolver,
		config:               config,
		statsReportInterval:  statsReportInterval,
		validator:            validator,
		p2pDialer:            p2pDialer,
		timeGetter:           time.Now,

		ratelimiter: rate.NewLimiter(rate.Every(1000*time.Millisecond), 1),
	}

	m.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, m.reconnectOnHold)

	return m
}

func (m *diagConnectionManager) chainID() int64 {
	return config.GetInt64(config.FlagChainID)
}

func (m *diagConnectionManager) HasConnection(providerID string) bool {
	m.connsMu.Lock()
	defer m.connsMu.Unlock()

	_, ok := m.conns[providerID]
	return ok
}

func (m *diagConnectionManager) GetReadyChan(providerID string) chan interface{} {
	m.connsMu.Lock()
	defer m.connsMu.Unlock()

	con, ok := m.conns[providerID]
	if ok {
		return con.resChannel
	}
	return nil
}

func (m *diagConnectionManager) Connect(consumerID identity.Identity, hermesID common.Address, proposalLookup ProposalLookup, params ConnectParams) (err error) {
	var sessionID session.ID

	ctx := context.Background()
	err = m.ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		log.Error().Msgf("ratelimiter.Wait: %s", err)
		return err
	}

	proposal, err := proposalLookup()
	if err != nil {
		return fmt.Errorf("failed to lookup proposal: %w", err)
	}

	tracer := trace.NewTracer("Consumer whole Connect")
	defer func() {
		traceResult := tracer.Finish(m.eventBus, string(sessionID))
		log.Debug().Msgf("Consumer connection trace: %s", traceResult)
	}()

	log.Error().Msgf("Connect > %v", proposal.ProviderID)
	uuid := proposal.ProviderID

	m.connsMu.Lock()
	con, ok := m.conns[uuid]
	if !ok {
		con = new(conn)
		con.status.State = connectionstate.NotConnected
		con.uuid = uuid
		m.conns[uuid] = con
	}
	m.connsMu.Unlock()

	removeConnection := func() {
		m.connsMu.Lock()
		defer m.connsMu.Unlock()
		delete(m.conns, uuid)
	}

	// make sure cache is cleared when connect terminates at any stage as part of disconnect
	// we assume that IPResolver might be used / cache IP before connect
	m.addCleanup(con, func() error {
		m.clearIPCache()
		return nil
	})

	if m.Status().State != connectionstate.NotConnected {
		removeConnection()
		return ErrAlreadyExists
	}

	prc := m.priceFromProposal(*proposal)

	err = m.validator.Validate(m.chainID(), consumerID, prc)
	if err != nil {
		removeConnection()
		return err
	}

	con.ctxLock.Lock()
	con.ctx, con.cancel = context.WithCancel(context.Background())
	con.ctxLock.Unlock()

	m.statusConnecting(con, consumerID, hermesID, *proposal)
	defer func() {
		if err != nil {
			log.Err(err).Msg("Connect failed, disconnecting")
			m.disconnect(con)
		}
	}()

	con.connectOptions = ConnectOptions{
		ConsumerID:     consumerID,
		HermesID:       hermesID,
		Proposal:       *proposal,
		ProposalLookup: proposalLookup,
		Params:         params,
	}

	con.activeConnection, err = m.newConnection(proposal.ServiceType)
	if err != nil {
		removeConnection()
		return err
	}

	sessionID, err = m.initSession(con, tracer, prc)
	if err != nil {
		removeConnection()
		return err
	}

	originalPublicIP := m.getPublicIP()

	err = m.startConnection(con, m.currentCtx(con), con.activeConnection, con.activeConnection.Start, con.connectOptions, tracer)
	if err != nil {
		removeConnection()
		return m.handleStartError(con, sessionID, err)
	}

	err = m.waitForConnectedState(con, con.activeConnection.State())
	if err != nil {
		removeConnection()
		return m.handleStartError(con, sessionID, err)
	}

	//m.statsTracker = newStatsTracker(m.eventBus, m.statsReportInterval)
	//go m.statsTracker.start(m, m.activeConnection)
	m.addCleanup(con, func() error {
		log.Trace().Msg("Cleaning: stopping statistics publisher")
		defer log.Trace().Msg("Cleaning: stopping statistics publisher DONE")
		//m.statsTracker.stop()

		removeConnection()
		return nil
	})

	con.resChannel = make(chan interface{})
	go Diag(m, con, proposal.ProviderID)

	go m.consumeConnectionStates(con, con.activeConnection.State())
	go m.checkSessionIP(con, con.channel, con.connectOptions.ConsumerID, con.connectOptions.SessionID, originalPublicIP)

	return nil
}

func (m *diagConnectionManager) autoReconnect(con *conn) (err error) {
	var sessionID session.ID

	tracer := trace.NewTracer("Consumer whole autoReconnect")
	defer func() {
		traceResult := tracer.Finish(m.eventBus, string(sessionID))
		log.Debug().Msgf("Consumer connection trace: %s", traceResult)
	}()

	proposal, err := con.connectOptions.ProposalLookup()
	if err != nil {
		return fmt.Errorf("failed to lookup proposal: %w", err)
	}

	con.connectOptions.Proposal = *proposal

	sessionID, err = m.initSession(con, tracer, m.priceFromProposal(con.connectOptions.Proposal))
	if err != nil {
		return err
	}

	err = m.startConnection(con, m.currentCtx(con), con.activeConnection, con.activeConnection.Reconnect, con.connectOptions, tracer)
	if err != nil {
		return m.handleStartError(con, sessionID, err)
	}

	return nil
}

func (m *diagConnectionManager) priceFromProposal(proposal proposal.PricedServiceProposal) market.Price {
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

func (m *diagConnectionManager) initSession(con *conn, tracer *trace.Tracer, prc market.Price) (sessionID session.ID, err error) {

	err = m.createP2PChannel(con, con.connectOptions, tracer)
	if err != nil {
		return sessionID, fmt.Errorf("could not create p2p channel during connect: %w", err)
	}

	con.connectOptions.ProviderNATConn = con.channel.ServiceConn()
	con.connectOptions.ChannelConn = con.channel.Conn()

	paymentSession, err := m.paymentLoop(con, con.connectOptions, prc)
	if err != nil {
		return sessionID, err
	}

	sessionDTO, err := m.createP2PSession(con, con.activeConnection, con.connectOptions, tracer, prc)
	sessionID = session.ID(sessionDTO.GetID())
	if err != nil {
		m.sendSessionStatus(con, con.channel, con.connectOptions.ConsumerID, sessionID, connectivity.StatusSessionEstablishmentFailed, err)
		return sessionID, err
	}

	traceStart := tracer.StartStage("Consumer session creation (start)")
	go m.keepAliveLoop(con, con.channel, sessionID)
	m.setStatus(con, func(status *connectionstate.Status) {
		status.SessionID = sessionID
	})
	m.publishSessionCreate(con, sessionID)
	paymentSession.SetSessionID(string(sessionID))
	tracer.EndStage(traceStart)

	con.connectOptions.SessionID = sessionID
	con.connectOptions.SessionConfig = sessionDTO.GetConfig()

	return sessionID, nil
}

func (m *diagConnectionManager) handleStartError(con *conn, sessionID session.ID, err error) error {

	if errors.Is(err, context.Canceled) {
		return ErrConnectionCancelled
	}
	m.addCleanupAfterDisconnect(con, func() error {
		return m.sendSessionStatus(con, con.channel, con.connectOptions.ConsumerID, sessionID, connectivity.StatusConnectionFailed, err)
	})
	m.publishStateEvent(con, connectionstate.StateConnectionFailed)

	log.Info().Err(err).Msg("Cancelling connection initiation: ")
	m.Cancel()
	return err
}

func (m *diagConnectionManager) clearIPCache() {
	if config.GetBool(config.FlagProxyMode) || config.GetBool(config.FlagDVPNMode) {
		return
	}

	if cr, ok := m.ipResolver.(*ip.CachedResolver); ok {
		cr.ClearCache()
	}
}

// checkSessionIP checks if IP has changed after connection was established.
func (m *diagConnectionManager) checkSessionIP(con *conn, channel p2p.Channel, consumerID identity.Identity, sessionID session.ID, originalPublicIP string) {
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
			m.sendSessionStatus(con, channel, consumerID, sessionID, connectivity.StatusConnectionOk, nil)
			return
		}

		// Notify peer and quality oracle that ip is not changed after tunnel connection was established.
		if i == m.config.IPCheck.MaxAttempts {
			m.sendSessionStatus(con, channel, consumerID, sessionID, connectivity.StatusSessionIPNotChanged, nil)
			m.publishStateEvent(con, connectionstate.StateIPNotChanged)
			return
		}

		time.Sleep(m.config.IPCheck.SleepDurationAfterCheck)
	}
}

// sendSessionStatus sends session connectivity status to other peer.
func (m *diagConnectionManager) sendSessionStatus(con *conn, channel p2p.ChannelSender, consumerID identity.Identity, sessionID session.ID, code connectivity.StatusCode, errDetails error) error {
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

	ctx, cancel := context.WithTimeout(m.currentCtx(con), 20*time.Second)
	defer cancel()
	_, err := channel.Send(ctx, p2p.TopicSessionStatus, p2p.ProtoMessage(sessionStatus))
	if err != nil {
		return fmt.Errorf("could not send p2p session status message: %w", err)
	}

	return nil
}

func (m *diagConnectionManager) getPublicIP() string {
	currentPublicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		log.Error().Err(err).Msg("Could not get current public IP")
		return ""
	}
	return currentPublicIP
}

func (m *diagConnectionManager) paymentLoop(con *conn, opts ConnectOptions, price market.Price) (PaymentIssuer, error) {

	payments, err := m.paymentEngineFactory(con.uuid, con.channel, opts.ConsumerID, identity.FromAddress(opts.Proposal.ProviderID), opts.HermesID, opts.Proposal, price)
	if err != nil {
		return nil, err
	}
	m.addCleanup(con, func() error {
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
				m.statusOnHold(con)
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

func (m *diagConnectionManager) cleanConnection(con *conn) {
	con.cleanupLock.Lock()
	defer con.cleanupLock.Unlock()

	for i := len(con.cleanup) - 1; i >= 0; i-- {
		log.Trace().Msgf("Connection cleaning up: (%v/%v)", i+1, len(con.cleanup))
		err := con.cleanup[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	con.cleanup = nil
}

func (m *diagConnectionManager) cleanAfterDisconnect(con *conn) {
	con.cleanupLock.Lock()
	defer con.cleanupLock.Unlock()

	for i := len(con.cleanupAfterDisconnect) - 1; i >= 0; i-- {
		log.Trace().Msgf("Connection cleaning up (after disconnect): (%v/%v)", i+1, len(con.cleanupAfterDisconnect))
		err := con.cleanupAfterDisconnect[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	con.cleanupAfterDisconnect = nil
}

func (m *diagConnectionManager) createP2PChannel(con *conn, opts ConnectOptions, tracer *trace.Tracer) error {
	trace := tracer.StartStage("Consumer P2P channel creation")
	defer tracer.EndStage(trace)

	contactDef, err := p2p.ParseContact(opts.Proposal.Contacts)
	if err != nil {
		return fmt.Errorf("provider does not support p2p communication: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(m.currentCtx(con), p2pDialTimeout)
	defer cancel()

	// TODO register all handlers before channel read/write loops
	channel, err := m.p2pDialer.Dial(timeoutCtx, opts.ConsumerID, identity.FromAddress(opts.Proposal.ProviderID), opts.Proposal.ServiceType, contactDef, tracer)
	if err != nil {
		return fmt.Errorf("p2p dialer failed: %w", err)
	}
	m.addCleanupAfterDisconnect(con, func() error {
		log.Trace().Msg("Cleaning: closing P2P communication channel")
		defer log.Trace().Msg("Cleaning: P2P communication channel DONE")

		return channel.Close()
	})

	con.channel = channel
	return nil
}

func (m *diagConnectionManager) addCleanupAfterDisconnect(con *conn, fn func() error) {
	con.cleanupLock.Lock()
	defer con.cleanupLock.Unlock()
	con.cleanupAfterDisconnect = append(con.cleanupAfterDisconnect, fn)
}

func (m *diagConnectionManager) addCleanup(con *conn, fn func() error) {
	con.cleanupLock.Lock()
	defer con.cleanupLock.Unlock()
	con.cleanup = append(con.cleanup, fn)
}

func (m *diagConnectionManager) createP2PSession(con *conn, c Connection, opts ConnectOptions, tracer *trace.Tracer, requestedPrice market.Price) (*pb.SessionResponse, error) {
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
	ctx, cancel := context.WithTimeout(m.currentCtx(con), 20*time.Second)
	defer cancel()
	res, err := con.channel.Send(ctx, p2p.TopicSessionCreate, p2p.ProtoMessage(sessionRequest))
	if err != nil {
		return nil, fmt.Errorf("could not send p2p session create request: %w", err)
	}

	var sessionResponse pb.SessionResponse
	err = res.UnmarshalProto(&sessionResponse)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal session reply to proto: %w", err)
	}

	channel := con.channel
	con.acknowledge = func() {
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
	m.addCleanupAfterDisconnect(con, func() error {
		log.Trace().Msg("Cleaning: requesting session destroy")
		defer log.Trace().Msg("Cleaning: requesting session destroy DONE")

		sessionDestroy := &pb.SessionInfo{
			ConsumerID: opts.ConsumerID.Address,
			SessionID:  sessionResponse.GetID(),
		}

		log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicSessionDestroy, sessionDestroy.String())
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := con.channel.Send(ctx, p2p.TopicSessionDestroy, p2p.ProtoMessage(sessionDestroy))
		if err != nil {
			return fmt.Errorf("could not send session destroy request: %w", err)
		}

		return nil
	})

	return &sessionResponse, nil
}

func (m *diagConnectionManager) publishSessionCreate(con *conn, sessionID session.ID) {
	sessionInfo := m.Status()
	// avoid printing IP address in logs
	sessionInfo.ConsumerLocation.IP = ""

	m.eventBus.Publish(connectionstate.AppTopicConnectionSession, connectionstate.AppEventConnectionSession{
		Status:      connectionstate.SessionCreatedStatus,
		SessionInfo: sessionInfo,
	})

	m.addCleanup(con, func() error {
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

func (m *diagConnectionManager) startConnection(con *conn, ctx context.Context, conn Connection, start ConnectionStart, connectOptions ConnectOptions, tracer *trace.Tracer) (err error) {
	trace := tracer.StartStage("Consumer start connection")
	defer tracer.EndStage(trace)

	if err = start(ctx, connectOptions); err != nil {
		return err
	}
	m.addCleanup(con, func() error {
		log.Trace().Msg("Cleaning: stopping connection")
		defer log.Trace().Msg("Cleaning: stopping connection DONE")
		conn.Stop()
		return nil
	})

	err = m.setupTrafficBlock(con, connectOptions.Params.DisableKillSwitch)
	if err != nil {
		return err
	}

	// Clear IP cache so session IP check can report that IP has really changed.
	m.clearIPCache()

	return nil
}

func (m *diagConnectionManager) Status() connectionstate.Status {
	log.Debug().Msg("Status() - not used")
	return connectionstate.Status{State: connectionstate.NotConnected}
}

func (m *diagConnectionManager) UUID() string {
	log.Debug().Msg("UUID() - not used")
	return ""
}

func (m *diagConnectionManager) Stats() connectionstate.Statistics {
	log.Debug().Msg("Stats() - not used")
	return connectionstate.Statistics{}
}

func (m *diagConnectionManager) setStatus(con *conn, delta func(status *connectionstate.Status)) {
	con.statusLock.Lock()
	stateWas := con.status.State

	delta(&con.status)

	state := con.status.State
	con.statusLock.Unlock()

	if state != stateWas {
		log.Info().Msgf("Connection state: %v -> %v", stateWas, state)
		m.publishStateEvent(con, state)
	}
}

func (m *diagConnectionManager) statusConnecting(con *conn, consumerID identity.Identity, accountantID common.Address, proposal proposal.PricedServiceProposal) {
	m.setStatus(con, func(status *connectionstate.Status) {
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

func (m *diagConnectionManager) statusConnected(con *conn) {
	m.setStatus(con, func(status *connectionstate.Status) {
		status.State = connectionstate.Connected
	})
}

func (m *diagConnectionManager) statusReconnecting(con *conn) {
	m.setStatus(con, func(status *connectionstate.Status) {
		status.State = connectionstate.Reconnecting
	})
}

func (m *diagConnectionManager) statusNotConnected(con *conn) {
	m.setStatus(con, func(status *connectionstate.Status) {
		status.State = connectionstate.NotConnected
	})
}

func (m *diagConnectionManager) statusDisconnecting(con *conn) {
	m.setStatus(con, func(status *connectionstate.Status) {
		status.State = connectionstate.Disconnecting
	})
}

func (m *diagConnectionManager) statusCanceled(con *conn) {
	m.setStatus(con, func(status *connectionstate.Status) {
		status.State = connectionstate.Canceled
	})
}

func (m *diagConnectionManager) statusOnHold(con *conn) {
	m.setStatus(con, func(status *connectionstate.Status) {
		status.State = connectionstate.StateOnHold
	})
}

func (m *diagConnectionManager) Cancel() {
	log.Error().Msg("Cancel() - not used")
}

func (m *diagConnectionManager) DisconnectSingle(con *conn) error {
	m.statusDisconnecting(con)
	m.disconnect(con)
	return nil
}

func (m *diagConnectionManager) Disconnect() error {
	log.Error().Msg("Disconnect() - not used")
	return nil
}

func (m *diagConnectionManager) CheckChannel(con *conn, ctx context.Context) error {
	if err := m.sendKeepAlivePing(ctx, con.channel, m.Status().SessionID); err != nil {
		return fmt.Errorf("keep alive ping failed: %w", err)
	}
	return nil
}

func (m *diagConnectionManager) disconnect(con *conn) {
	con.discoLock.Lock()
	defer con.discoLock.Unlock()

	con.cleanupFinishedLock.Lock()
	defer con.cleanupFinishedLock.Unlock()
	con.cleanupFinished = make(chan struct{})
	defer close(con.cleanupFinished)

	con.ctxLock.Lock()
	con.cancel()
	con.ctxLock.Unlock()

	m.cleanConnection(con)
	m.statusNotConnected(con)

	m.cleanAfterDisconnect(con)
}

func (m *diagConnectionManager) waitForConnectedState(con *conn, stateChannel <-chan connectionstate.State) error {
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
				if con.acknowledge != nil {
					go con.acknowledge()
				}
				m.onStateChanged(con, state)
				return nil
			default:
				m.onStateChanged(con, state)
			}
		case <-m.currentCtx(con).Done():
			return m.currentCtx(con).Err()
		}
	}
}

func (m *diagConnectionManager) consumeConnectionStates(con *conn, stateChannel <-chan connectionstate.State) {
	for state := range stateChannel {
		m.onStateChanged(con, state)
	}
}

func (m *diagConnectionManager) onStateChanged(con *conn, state connectionstate.State) {
	log.Debug().Msgf("Connection state received: %s", state)

	// React just to certain stains from connection. Because disconnect happens in connectionWaiter
	switch state {
	case connectionstate.Connected:
		m.statusConnected(con)
	case connectionstate.Reconnecting:
		m.statusReconnecting(con)
	}
}

func (m *diagConnectionManager) setupTrafficBlock(con *conn, disableKillSwitch bool) error {
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
	m.addCleanup(con, func() error {
		log.Trace().Msg("Cleaning: traffic block rule")
		defer log.Trace().Msg("Cleaning: traffic block rule DONE")

		removeRule()

		return nil
	})
	return nil
}

func (m *diagConnectionManager) reconnectOnHold(state connectionstate.AppEventConnectionState) {
	if state.State != connectionstate.StateOnHold || !config.GetBool(config.FlagAutoReconnect) {
		return
	}

	con, ok := m.conns[state.UUID]
	if !ok {
		return
	}

	if con.channel != nil {
		con.channel.Close()
	}

	con.preReconnect()
	m.clearIPCache()

	for err := m.autoReconnect(con); err != nil; err = m.autoReconnect(con) {
		select {
		case <-m.currentCtx(con).Done():
			log.Info().Err(m.currentCtx(con).Err()).Msg("Stopping reconnect")
			return
		default:
			log.Error().Err(err).Msg("Failed to reconnect active session, will try again")
		}
	}
	con.postReconnect()
}

func (m *diagConnectionManager) publishStateEvent(con *conn, state connectionstate.State) {
	sessionInfo := m.Status()
	// avoid printing IP address in logs
	sessionInfo.ConsumerLocation.IP = ""

	m.eventBus.Publish(connectionstate.AppTopicConnectionState, connectionstate.AppEventConnectionState{
		UUID:        con.uuid,
		State:       state,
		SessionInfo: sessionInfo,
	})
}

func (m *diagConnectionManager) keepAliveLoop(con *conn, channel p2p.Channel, sessionID session.ID) {
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
		case <-m.currentCtx(con).Done():
			log.Debug().Msgf("Stopping p2p keepalive: %v", m.currentCtx(con).Err())
			return
		case <-time.After(m.config.KeepAlive.SendInterval):
			ctx, cancel := context.WithTimeout(context.Background(), m.config.KeepAlive.SendTimeout)
			if err := m.sendKeepAlivePing(ctx, channel, sessionID); err != nil {
				log.Err(err).Msgf("Failed to send p2p keepalive ping. SessionID=%s", sessionID)
				errCount++
				if errCount == m.config.KeepAlive.MaxSendErrCount {
					log.Error().Msgf("Max p2p keepalive err count reached, disconnecting. SessionID=%s", sessionID)
					if config.GetBool(config.FlagKeepConnectedOnFail) {
						m.statusOnHold(con)
					} else {
						//m.Disconnect()
						log.Error().Msgf("Max p2p keepalive err count reached, disconnecting. SessionID=%s >>>>>>>>>", sessionID)
						m.DisconnectSingle(con)
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

func (m *diagConnectionManager) sendKeepAlivePing(ctx context.Context, channel p2p.Channel, sessionID session.ID) error {
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

func (m *diagConnectionManager) currentCtx(con *conn) context.Context {
	con.ctxLock.RLock()
	defer con.ctxLock.RUnlock()

	return con.ctx
}

func (m *diagConnectionManager) Reconnect() {
	log.Error().Msg("Reconnect - not used")
}
