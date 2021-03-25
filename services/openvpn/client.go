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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/auth"
	openvpn_bytescount "github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/bytescount"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

// ErrProcessNotStarted represents the error we return when the process is not started yet
var ErrProcessNotStarted = errors.New("process not started yet")

// processFactory creates a new openvpn process
type processFactory func(options connection.ConnectOptions, sessionConfig VPNConfig) (openvpn.Process, *ClientConfig, error)

// NewClient creates a new openvpn connection
func NewClient(openvpnBinary, scriptDir, runtimeDir string,
	signerFactory identity.SignerFactory,
	ipResolver ip.Resolver,
) (connection.Connection, error) {
	stateCh := make(chan connectionstate.State, 100)
	client := &Client{
		scriptDir:           scriptDir,
		runtimeDir:          runtimeDir,
		signerFactory:       signerFactory,
		stateCh:             stateCh,
		ipResolver:          ipResolver,
		removeAllowedIPRule: func() {},
	}

	procFactory := func(options connection.ConnectOptions, sessionConfig VPNConfig) (openvpn.Process, *ClientConfig, error) {
		vpnClientConfig, err := NewClientConfigFromSession(sessionConfig, scriptDir, runtimeDir, options)
		if err != nil {
			return nil, nil, err
		}

		signer := signerFactory(options.ConsumerID)

		stateMiddleware := newStateMiddleware(stateCh)
		authMiddleware := newAuthMiddleware(options.SessionID, signer)
		byteCountMiddleware := openvpn_bytescount.NewMiddleware(client.OnStats, connection.DefaultStatsReportInterval)
		proc := openvpn.CreateNewProcess(openvpnBinary, vpnClientConfig.GenericConfig, stateMiddleware, byteCountMiddleware, authMiddleware)
		return proc, vpnClientConfig, nil
	}

	client.processFactory = procFactory
	return client, nil
}

// Client takes in the openvpn process and works with it
type Client struct {
	scriptDir           string
	runtimeDir          string
	signerFactory       identity.SignerFactory
	stateCh             chan connectionstate.State
	stats               connectionstate.Statistics
	statsMu             sync.RWMutex
	process             openvpn.Process
	processFactory      processFactory
	ipResolver          ip.Resolver
	removeAllowedIPRule func()
	stopOnce            sync.Once
}

var _ connection.Connection = &Client{}

// State returns connection state channel.
func (c *Client) State() <-chan connectionstate.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Client) Statistics() (connectionstate.Statistics, error) {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats, nil
}

func (c *Client) Reconnect(ctx context.Context, options connection.ConnectOptions) error {
	return fmt.Errorf("not supported")
}

// Start starts the connection
func (c *Client) Start(ctx context.Context, options connection.ConnectOptions) error {
	log.Info().Msg("Starting connection")

	sessionConfig := VPNConfig{}
	err := json.Unmarshal(options.SessionConfig, &sessionConfig)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal session config")
	}

	c.removeAllowedIPRule, err = firewall.AllowIPAccess(sessionConfig.RemoteIP)
	if err != nil {
		return errors.Wrap(err, "failed to add allowed IP address")
	}

	proc, clientConfig, err := c.processFactory(options, sessionConfig)
	if err != nil {
		log.Info().Err(err).Msg("Client config factory error")
		return errors.Wrap(err, "client config factory error")
	}
	c.process = proc
	log.Info().Interface("data", clientConfig).Msgf("Openvpn client configuration")

	err = c.process.Start()
	if err != nil {
		c.removeAllowedIPRule()
	}
	return errors.Wrap(err, "failed to start client process")
}

// Stop stops the connection
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		if c.process != nil {
			c.process.Stop()
		}
		c.removeAllowedIPRule()
	})
}

// OnStats updates connection statistics.
func (c *Client) OnStats(cnt openvpn_bytescount.Bytecount) error {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.At = time.Now()
	c.stats.BytesReceived = cnt.BytesIn
	c.stats.BytesSent = cnt.BytesOut
	return nil
}

// GetConfig returns the consumer-side configuration.
func (c *Client) GetConfig() (connection.ConsumerConfig, error) {
	return &ConsumerConfig{}, nil
}

// VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	DNSIPs          string `json:"dns_ips"`
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	LocalPort       int    `json:"lport"`
	Ports           []int  `json:"ports"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}

func newAuthMiddleware(sessionID session.ID, signer identity.Signer) management.Middleware {
	credentialsProvider := SignatureCredentialsProvider(sessionID, signer)
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
		if connectionState != connectionstate.Unknown {
			stateChannel <- connectionState
		}

		// this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if openvpnState == openvpn.ProcessExited {
			close(stateChannel)
		}
	}
}

// openvpnStateMap maps openvpn states to connection state
var openvpnStateMap = map[openvpn.State]connectionstate.State{
	openvpn.ConnectedState:    connectionstate.Connected,
	openvpn.ExitingState:      connectionstate.Disconnecting,
	openvpn.ReconnectingState: connectionstate.Reconnecting,
}

// openVpnStateCallbackToConnectionState maps openvpn.State to connection.State. Returns a pointer to connection.state, or nil
func openVpnStateCallbackToConnectionState(input openvpn.State) connectionstate.State {
	if val, ok := openvpnStateMap[input]; ok {
		return val
	}
	return connectionstate.Unknown
}
