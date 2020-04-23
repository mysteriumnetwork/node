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
	"sync"
	"time"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/auth"
	openvpn_bytescount "github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/bytescount"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrProcessNotStarted represents the error we return when the process is not started yet
var ErrProcessNotStarted = errors.New("process not started yet")

// processFactory creates a new openvpn process
type processFactory func(options connection.ConnectOptions, sessionConfig VPNConfig) (openvpn.Process, *ClientConfig, error)

type natPinger interface {
	PingProvider(ctx context.Context, ip string, localPorts, remotePorts []int, proxyPort int) (localPort, remotePort int, err error)
}

// NewClient creates a new openvpn connection
func NewClient(openvpnBinary, configDirectory, runtimeDirectory string,
	signerFactory identity.SignerFactory,
	ipResolver ip.Resolver,
	natPinger natPinger,
) (connection.Connection, error) {

	stateCh := make(chan connection.State, 100)
	client := &Client{
		configDirectory:     configDirectory,
		runtimeDirectory:    runtimeDirectory,
		signerFactory:       signerFactory,
		stateCh:             stateCh,
		ipResolver:          ipResolver,
		natPinger:           natPinger,
		removeAllowedIPRule: func() {},
	}

	procFactory := func(options connection.ConnectOptions, sessionConfig VPNConfig) (openvpn.Process, *ClientConfig, error) {
		vpnClientConfig, err := NewClientConfigFromSession(sessionConfig, configDirectory, runtimeDirectory, options)
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
	openvpnBinary       string
	configDirectory     string
	runtimeDirectory    string
	signerFactory       identity.SignerFactory
	stateCh             chan connection.State
	stats               connection.Statistics
	statsMu             sync.RWMutex
	process             openvpn.Process
	processFactory      processFactory
	ipResolver          ip.Resolver
	natPinger           natPinger
	ports               []int
	removeAllowedIPRule func()
	stopOnce            sync.Once
}

var _ connection.Connection = &Client{}

// State returns connection state channel.
func (c *Client) State() <-chan connection.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Client) Statistics() (connection.Statistics, error) {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats, nil
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

	// TODO this backward compatibility block needs to be removed once we will fully migrate to the p2p communication.
	if len(sessionConfig.Ports) > 0 {
		ip := sessionConfig.RemoteIP
		lPort, rPort, err := c.natPinger.PingProvider(ctx, ip, c.ports, sessionConfig.Ports, sessionConfig.LocalPort)
		if err != nil {
			c.removeAllowedIPRule()
			return errors.Wrap(err, "could not ping provider")
		}

		sessionConfig.LocalPort = lPort
		sessionConfig.RemotePort = rPort
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

	ports, err := port.NewPool().AcquireMultiple(config.GetInt(config.FlagNATPunchingMaxTTL))
	if err != nil {
		return nil, err
	}

	for _, p := range ports {
		c.ports = append(c.ports, p.Num())
	}

	return &ConsumerConfig{
		IP:    publicIP,
		Ports: c.ports,
	}, nil
}

//VPNConfig structure represents VPN configuration options for given session
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
