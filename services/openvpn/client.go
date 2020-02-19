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

package openvpn

import (
	"encoding/json"
	"sync"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/auth"
	openvpn_bytescount "github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/bytescount"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrProcessNotStarted represents the error we return when the process is not started yet
var ErrProcessNotStarted = errors.New("process not started yet")

// processFactory creates a new openvpn process
type processFactory func(options connection.ConnectOptions) (openvpn.Process, *ClientConfig, error)

// NewClient creates a new openvpn connection
func NewClient(openvpnBinary, configDirectory, runtimeDirectory string,
	signerFactory identity.SignerFactory,
	ipResolver ip.Resolver,
	natPinger traversal.NATProviderPinger,
) (connection.Connection, error) {

	stateCh := make(chan connection.State, 100)
	client := &Client{
		configDirectory:     configDirectory,
		runtimeDirectory:    runtimeDirectory,
		signerFactory:       signerFactory,
		stateCh:             stateCh,
		ipResolver:          ipResolver,
		natPinger:           natPinger,
		pingerStop:          make(chan struct{}),
		removeAllowedIPRule: func() {},
	}

	procFactory := func(options connection.ConnectOptions) (openvpn.Process, *ClientConfig, error) {
		sessionConfig := &VPNConfig{}
		err := json.Unmarshal(options.SessionConfig, sessionConfig)
		if err != nil {
			return nil, nil, err
		}

		// override vpnClientConfig params with proxy local IP and pinger port
		// do this only if connecting to natted provider
		if sessionConfig.LocalPort > 0 {
			sessionConfig.OriginalRemoteIP = sessionConfig.RemoteIP
			sessionConfig.OriginalRemotePort = sessionConfig.RemotePort
		}

		vpnClientConfig, err := NewClientConfigFromSession(sessionConfig, configDirectory, runtimeDirectory, options.DNS)
		if err != nil {
			return nil, nil, err
		}

		signer := signerFactory(options.ConsumerID)

		stateMiddleware := newStateMiddleware(stateCh)
		authMiddleware := newAuthMiddleware(options.SessionID, signer)
		byteCountMiddleware := openvpn_bytescount.NewMiddleware(client.OnStats, connection.StatsReportInterval)
		proc := openvpn.CreateNewProcess(openvpnBinary, vpnClientConfig.GenericConfig, stateMiddleware, byteCountMiddleware, authMiddleware)
		return proc, vpnClientConfig, nil
	}

	client.processFactory = procFactory
	return client, nil
}

// Client takes in the openvpn process and works with it
type Client struct {
	openvpnBinary       string
	configDirectory     string
	runtimeDirectory    string
	signerFactory       identity.SignerFactory
	stateCh             chan connection.State
	stats               consumer.SessionStatistics
	statsMu             sync.RWMutex
	process             openvpn.Process
	processFactory      processFactory
	ipResolver          ip.Resolver
	natPinger           traversal.NATProviderPinger
	pingerStop          chan struct{}
	removeAllowedIPRule func()
	stopOnce            sync.Once
}

var _ connection.Connection = &Client{}

// State returns connection state channel.
func (c *Client) State() <-chan connection.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Client) Statistics() (consumer.SessionStatistics, error) {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats, nil
}

// Start starts the connection
func (c *Client) Start(options connection.ConnectOptions) error {
	log.Info().Msg("Starting connection")
	proc, clientConfig, err := c.processFactory(options)
	if err != nil {
		log.Info().Err(err).Msg("Client config factory error")
		return err
	}
	c.process = proc
	log.Info().Interface("data", clientConfig).Msgf("Openvpn client configuration")
	removeAllowedIPRule, err := firewall.AllowIPAccess(clientConfig.VpnConfig.RemoteIP)
	if err != nil {
		return err
	}
	c.removeAllowedIPRule = removeAllowedIPRule

	if clientConfig.VpnConfig.LocalPort > 0 {
		err = c.natPinger.PingProvider(
			clientConfig.VpnConfig.OriginalRemoteIP,
			clientConfig.VpnConfig.OriginalRemotePort,
			clientConfig.LocalPort,
			clientConfig.LocalPort+1,
			c.pingerStop,
		)
		if err != nil {
			removeAllowedIPRule()
			return err
		}
	}
	err = c.process.Start()
	if err != nil {
		removeAllowedIPRule()
	}
	return err
}

// Wait waits for the connection to exit
func (c *Client) Wait() error {
	if c.process == nil {
		return ErrProcessNotStarted
	}
	return c.process.Wait()
}

// Stop stops the connection
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		if c.process != nil {
			c.process.Stop()
		}
		c.removeAllowedIPRule()
		close(c.pingerStop)
	})
}

// OnStats updates connection statistics.
func (c *Client) OnStats(cnt openvpn_bytescount.Bytecount) error {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.BytesReceived = cnt.BytesIn
	c.stats.BytesSent = cnt.BytesOut
	return nil
}

// GetConfig returns the consumer-side configuration.
func (c *Client) GetConfig() (connection.ConsumerConfig, error) {
	switch c.natPinger.(type) {
	case *traversal.NoopPinger:
		log.Info().Msg("Noop pinger detected, returning nil client config.")
		return nil, nil
	}

	publicIP, err := c.ipResolver.GetPublicIP()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get consumer public IP")
	}

	return &ConsumerConfig{
		IP: publicIP,
	}, nil
}

//VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	// OriginalRemoteIP and OriginalRemotePort are used for NAT punching from consumer side.
	OriginalRemoteIP   string
	OriginalRemotePort int

	DNSIPs          string `json:"dns_ips"`
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	LocalPort       int    `json:"lport"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}

func newAuthMiddleware(sessionID session.ID, signer identity.Signer) management.Middleware {
	credentialsProvider := openvpn_session.SignatureCredentialsProvider(sessionID, signer)
	return auth.NewMiddleware(credentialsProvider)
}

func newStateMiddleware(stateChannel connection.StateChannel) management.Middleware {
	stateCallback := getStateCallback(stateChannel)
	return state.NewMiddleware(stateCallback)
}

// getStateCallback returns the callback for working with openvpn state
func getStateCallback(stateChannel connection.StateChannel) func(openvpnState openvpn.State) {
	return func(openvpnState openvpn.State) {
		connectionState := openVpnStateCallbackToConnectionState(openvpnState)
		if connectionState != connection.Unknown {
			stateChannel <- connectionState
		}

		//this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if openvpnState == openvpn.ProcessExited {
			close(stateChannel)
		}
	}
}

// openvpnStateMap maps openvpn states to connection state
var openvpnStateMap = map[openvpn.State]connection.State{
	openvpn.ConnectedState:    connection.Connected,
	openvpn.ExitingState:      connection.Disconnecting,
	openvpn.ReconnectingState: connection.Reconnecting,
}

// openVpnStateCallbackToConnectionState maps openvpn.State to connection.State. Returns a pointer to connection.state, or nil
func openVpnStateCallbackToConnectionState(input openvpn.State) connection.State {
	if val, ok := openvpnStateMap[input]; ok {
		return val
	}
	return connection.Unknown
}
