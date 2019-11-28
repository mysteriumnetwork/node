/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package mysterium

import (
	"encoding/json"
	"sync"

	"github.com/mysteriumnetwork/go-openvpn/openvpn3"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/session"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type natPinger interface {
	PingProvider(ip string, port int, consumerPort int, stop <-chan struct{}) error
	StopNATProxy()
	SetProtectSocketCallback(SocketProtect func(socket int) bool)
}

type openvpn3SessionFactory func(connection.ConnectOptions) (*openvpn3.Session, *openvpn.ClientConfig, error)

var errSessionWrapperNotStarted = errors.New("session wrapper not started")

type openvpnConnection struct {
	session       *openvpn3.Session
	createSession openvpn3SessionFactory
	natPinger     natPinger
	ipResolver    ip.Resolver
	pingerStop    chan struct{}
	stopOnce      sync.Once
}

func (wrapper *openvpnConnection) Start(options connection.ConnectOptions) error {
	newSession, clientConfig, err := wrapper.createSession(options)
	if err != nil {
		return err
	}

	log.Info().Msgf("Client config after session create: %v", clientConfig)
	if clientConfig.LocalPort > 0 {
		err := wrapper.natPinger.PingProvider(
			clientConfig.VpnConfig.OriginalRemoteIP,
			clientConfig.VpnConfig.OriginalRemotePort,
			clientConfig.LocalPort,
			wrapper.pingerStop,
		)
		if err != nil {
			return err
		}
	}

	wrapper.session = newSession
	wrapper.session.Start()
	return nil
}

func (wrapper *openvpnConnection) Stop() {
	wrapper.stopOnce.Do(func() {
		if wrapper.session != nil {
			wrapper.session.Stop()
		}
		log.Info().Msg("Stopping NATProxy")
		close(wrapper.pingerStop)
	})
}

func (wrapper *openvpnConnection) Wait() error {
	if wrapper.session != nil {
		return wrapper.session.Wait()
	}
	return errSessionWrapperNotStarted
}

func (wrapper *openvpnConnection) GetConfig() (connection.ConsumerConfig, error) {
	ip, err := wrapper.ipResolver.GetPublicIP()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get consumer config")
	}

	switch wrapper.natPinger.(type) {
	case *traversal.NoopPinger:
		log.Info().Msg("Noop pinger detected, returning nil client config.")
		return nil, nil
	}

	return &openvpn.ConsumerConfig{
		// skip sending port here, since provider generates port for consumer in VPNConfig
		IP: &ip,
	}, nil
}

func channelToCallbacks(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) openvpn3.MobileSessionCallbacks {
	return channelToCallbacksAdapter{
		stateChannel:      stateChannel,
		statisticsChannel: statisticsChannel,
	}
}

type channelToCallbacksAdapter struct {
	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel
}

func (adapter channelToCallbacksAdapter) OnEvent(event openvpn3.Event) {
	switch event.Name {
	case "CONNECTING":
		adapter.stateChannel <- connection.Connecting
	case "CONNECTED":
		adapter.stateChannel <- connection.Connected
	case "DISCONNECTED":
		adapter.stateChannel <- connection.Disconnecting
		adapter.stateChannel <- connection.NotConnected
		close(adapter.stateChannel)
		close(adapter.statisticsChannel)
	default:
		log.Info().Msgf("Unhandled event: %+v", event)
	}
}

func (channelToCallbacksAdapter) Log(text string) {
	log.Info().Msg("Openvpn log: " + text)
}

func (adapter channelToCallbacksAdapter) OnStats(openvpnStats openvpn3.Statistics) {
	sessionStats := consumer.SessionStatistics{
		BytesSent:     uint64(openvpnStats.BytesOut),
		BytesReceived: uint64(openvpnStats.BytesIn),
	}
	select {
	case adapter.statisticsChannel <- sessionStats:
	default:
		log.Warn().Msg("Statistics dropped. Channel full")
	}
}

// Openvpn3TunnelSetup is alias for openvpn3 tunnel setup interface exposed to Android/iOS interop
type Openvpn3TunnelSetup openvpn3.TunnelSetup

// ReconnectableSession interface exposing reconnect for an active session
type ReconnectableSession interface {
	Reconnect(afterSeconds int) error
}

type sessionTracker struct {
	session *openvpn3.Session
	mux     sync.Mutex
}

func (st *sessionTracker) sessionCreated(s *openvpn3.Session) {
	st.mux.Lock()
	st.session = s
	st.mux.Unlock()
}

// Reconnect reconnects active session after given time
func (st *sessionTracker) Reconnect(afterSeconds int) error {
	st.mux.Lock()
	defer st.mux.Unlock()
	if st.session == nil {
		return errors.New("session not created yet")
	}

	return st.session.Reconnect(afterSeconds)
}

func (st *sessionTracker) handleState(stateEvent connection.StateEvent) {
	// On disconnected - remove session
	if stateEvent.State == connection.Disconnecting {
		st.mux.Lock()
		st.session = nil
		st.mux.Unlock()
	}
}

// OpenvpnConnectionFactory is the connection factory for openvpn
type OpenvpnConnectionFactory struct {
	sessionTracker *sessionTracker
	signerFactory  identity.SignerFactory
	tunnelSetup    Openvpn3TunnelSetup
	natPinger      natPinger
	ipResolver     ip.Resolver
}

// Create creates a new openvpn connection
func (ocf *OpenvpnConnectionFactory) Create(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (con connection.Connection, err error) {
	sessionFactory := func(options connection.ConnectOptions) (*openvpn3.Session, *openvpn.ClientConfig, error) {
		sessionConfig := &openvpn.VPNConfig{}
		err := json.Unmarshal(options.SessionConfig, sessionConfig)
		if err != nil {
			return nil, nil, err
		}

		// override vpnClientConfig params with proxy local IP and pinger port
		// do this only if connecting to natted provider
		if sessionConfig.LocalPort > 0 {
			sessionConfig.OriginalRemoteIP = sessionConfig.RemoteIP
			sessionConfig.OriginalRemotePort = sessionConfig.RemotePort
			sessionConfig.RemoteIP = "127.0.0.1"
			// TODO: randomize this too?
			sessionConfig.RemotePort = sessionConfig.LocalPort + 1
		}

		vpnClientConfig, err := openvpn.NewClientConfigFromSession(sessionConfig, "", "", false)
		if err != nil {
			return nil, nil, err
		}

		log.Info().Msgf("Client config on create: %v", vpnClientConfig)

		profileContent, err := vpnClientConfig.ToConfigFileContent()
		if err != nil {
			return nil, nil, err
		}

		config := openvpn3.NewConfig(profileContent)
		config.GuiVersion = "govpn 0.1"
		config.CompressionMode = "asym"
		config.ConnTimeout = 0 //essentially means - reconnect forever (but can always stopped with disconnect)

		signer := ocf.signerFactory(options.ConsumerID)

		username, password, err := session.SignatureCredentialsProvider(options.SessionID, signer)()
		if err != nil {
			return nil, nil, err
		}

		credentials := openvpn3.UserCredentials{
			Username: username,
			Password: password,
		}

		ocf.natPinger.SetProtectSocketCallback(ocf.tunnelSetup.SocketProtect)
		newSession := openvpn3.NewMobileSession(config, credentials, channelToCallbacks(stateChannel, statisticsChannel), ocf.tunnelSetup)
		ocf.sessionTracker.sessionCreated(newSession)
		return newSession, vpnClientConfig, nil
	}
	return &openvpnConnection{
		createSession: sessionFactory,
		natPinger:     ocf.natPinger,
		ipResolver:    ocf.ipResolver,
		pingerStop:    make(chan struct{}),
	}, nil
}
