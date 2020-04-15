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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/mobile/mysterium/openvpn3"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/session"
)

const natPunchingMaxTTL = 10

type natPinger interface {
	traversal.NATProviderPinger
	SetProtectSocketCallback(SocketProtect func(socket int) bool)
}

type openvpn3SessionFactory func(connection.ConnectOptions, openvpn.VPNConfig) (*openvpn3.Session, *openvpn.ClientConfig, error)

var errSessionWrapperNotStarted = errors.New("session wrapper not started")

// NewOpenVPNConnection creates a new openvpn connection
func NewOpenVPNConnection(sessionTracker *sessionTracker, signerFactory identity.SignerFactory, tunnelSetup Openvpn3TunnelSetup, natPinger natPinger, ipResolver ip.Resolver) (con connection.Connection, err error) {
	conn := &openvpnConnection{
		stateCh:    make(chan connection.State, 100),
		natPinger:  natPinger,
		ipResolver: ipResolver,
	}

	sessionFactory := func(options connection.ConnectOptions, sessionConfig openvpn.VPNConfig) (*openvpn3.Session, *openvpn.ClientConfig, error) {
		vpnClientConfig, err := openvpn.NewClientConfigFromSession(sessionConfig, "", "", options)
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

		signer := signerFactory(options.ConsumerID)

		username, password, err := session.SignatureCredentialsProvider(options.SessionID, signer)()
		if err != nil {
			return nil, nil, err
		}

		credentials := openvpn3.UserCredentials{
			Username: username,
			Password: password,
		}

		newSession := openvpn3.NewMobileSession(config, credentials, conn, tunnelSetup)
		sessionTracker.sessionCreated(newSession)
		return newSession, vpnClientConfig, nil
	}
	conn.createSession = sessionFactory
	conn.tunnelSetup = tunnelSetup
	return conn, nil
}

type openvpnConnection struct {
	ports         []int
	stateCh       chan connection.State
	stats         connection.Statistics
	tunnelSetup   Openvpn3TunnelSetup
	statsMu       sync.RWMutex
	session       *openvpn3.Session
	createSession openvpn3SessionFactory
	natPinger     natPinger
	ipResolver    ip.Resolver
	stopOnce      sync.Once
	stopProxy     chan struct{}
}

var _ connection.Connection = &openvpnConnection{}

func (c *openvpnConnection) OnEvent(event openvpn3.Event) {
	switch event.Name {
	case "CONNECTING":
		c.stateCh <- connection.Connecting
	case "CONNECTED":
		c.stateCh <- connection.Connected
	case "DISCONNECTED":
		c.stateCh <- connection.Disconnecting
		c.stateCh <- connection.NotConnected
		close(c.stateCh)
	default:
		log.Info().Msgf("Unhandled event: %+v", event)
	}
}

func (c *openvpnConnection) OnStats(openvpnStats openvpn3.Statistics) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats = connection.Statistics{
		At:            time.Now(),
		BytesSent:     openvpnStats.BytesOut,
		BytesReceived: openvpnStats.BytesIn,
	}
}

func (c *openvpnConnection) Log(text string) {
	log.Info().Msg("Openvpn log: " + text)
}

func (c *openvpnConnection) State() <-chan connection.State {
	return c.stateCh
}

func (c *openvpnConnection) Statistics() (connection.Statistics, error) {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats, nil
}

func (c *openvpnConnection) Start(ctx context.Context, options connection.ConnectOptions) error {
	sessionConfig := openvpn.VPNConfig{}
	err := json.Unmarshal(options.SessionConfig, &sessionConfig)
	if err != nil {
		return err
	}

	if options.ProviderNATConn != nil {
		port, err := port.NewPool().Acquire()
		if err != nil {
			return errors.Wrap(err, "failed to acquire free port")
		}

		sessionConfig.LocalPort = port.Num()
		sessionConfig.RemoteIP = "127.0.0.1"
		sessionConfig.RemotePort = sessionConfig.LocalPort

		// Exclude p2p channel traffic from VPN tunnel.
		channelSocket, err := peekLookAtSocketFd4From(options.ChannelConn)
		if err != nil {
			return fmt.Errorf("could not get channel socket: %w", err)
		}

		proxy := traversal.NewNATProxy()

		c.tunnelSetup.SocketProtect(channelSocket)
		proxy.SetProtectSocketCallback(c.tunnelSetup.SocketProtect)

		localAddr := fmt.Sprintf("127.0.0.1:%d", sessionConfig.LocalPort)
		c.stopProxy = proxy.ConsumerHandOff(localAddr, options.ProviderNATConn)
	} else if len(sessionConfig.Ports) > 0 { // TODO this backward compatibility block needs to be removed once we will fully migrate to the p2p communication.
		if len(sessionConfig.Ports) == 0 || len(c.ports) == 0 {
			c.ports = []int{sessionConfig.LocalPort}
			sessionConfig.Ports = []int{sessionConfig.RemotePort}
		}

		if sessionConfig.LocalPort == 0 {
			lport, err := port.NewPool().Acquire()
			if err != nil {
				return errors.Wrap(err, "failed to acquire free port")
			}

			sessionConfig.LocalPort = lport.Num()
		}

		c.natPinger.SetProtectSocketCallback(c.tunnelSetup.SocketProtect)

		remoteIP := sessionConfig.RemoteIP
		_, _, err = c.natPinger.PingProvider(ctx, remoteIP, c.ports, sessionConfig.Ports, sessionConfig.LocalPort)
		if err != nil {
			return errors.Wrap(err, "could not ping provider")
		}

		sessionConfig.RemoteIP = "127.0.0.1"
		sessionConfig.RemotePort = sessionConfig.LocalPort
	}

	newSession, clientConfig, err := c.createSession(options, sessionConfig)
	if err != nil {
		log.Info().Err(err).Msg("Client config factory error")
		return err
	}
	log.Info().Interface("data", clientConfig).Msgf("Openvpn client configuration")

	c.session = newSession
	c.session.Start()
	return nil
}

func (c *openvpnConnection) Stop() {
	c.stopOnce.Do(func() {
		if c.session != nil {
			c.session.Stop()
		}

		if c.stopProxy != nil {
			close(c.stopProxy)
		}
	})
}

func (c *openvpnConnection) Wait() error {
	if c.session != nil {
		return c.session.Wait()
	}
	return errSessionWrapperNotStarted
}

func (c *openvpnConnection) GetConfig() (connection.ConsumerConfig, error) {
	// TODO the whole content of this function needs to be removed once we will migrate to the p2p communication.
	switch c.natPinger.(type) {
	case *traversal.NoopPinger:
		log.Info().Msg("Noop pinger detected, returning nil client config.")
		return nil, nil
	}

	publicIP, err := c.ipResolver.GetPublicIP()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get consumer public IP")
	}

	ports, err := port.NewPool().AcquireMultiple(natPunchingMaxTTL)
	if err != nil {
		return nil, err
	}

	for _, p := range ports {
		c.ports = append(c.ports, p.Num())
	}

	return &openvpn.ConsumerConfig{
		IP:    publicIP,
		Ports: c.ports,
	}, nil
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

func (st *sessionTracker) handleState(stateEvent connection.AppEventConnectionState) {
	// On disconnected - remove session
	if stateEvent.State == connection.Disconnecting {
		st.mux.Lock()
		st.session = nil
		st.mux.Unlock()
	}
}
