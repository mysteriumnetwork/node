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

package openvpn

import (
	"encoding/json"
	"time"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/auth"
	openvpn_bytescount "github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/bytescount"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/services/openvpn/middlewares/client/bytescount"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
	"github.com/mysteriumnetwork/node/session"
)

// OpenvpnProcessBasedConnectionFactory represents a factory for creating process-based openvpn connections
type OpenvpnProcessBasedConnectionFactory struct {
	mysteriumAPIClient    server.Client
	openvpnBinary         string
	configDirectory       string
	runtimeDirectory      string
	originalLocationCache location.Cache
	signerFactory         identity.SignerFactory
	statsKeeper           stats.SessionStatsKeeper
}

// NewOpenvpnProcessBasedConnectionFactory creates a new OpenvpnProcessBasedConnectionFactory
func NewOpenvpnProcessBasedConnectionFactory(
	mysteriumAPIClient server.Client,
	openvpnBinary, configDirectory, runtimeDirectory string,
	statsKeeper stats.SessionStatsKeeper,
	originalLocationCache location.Cache,
	signerFactory identity.SignerFactory,
) *OpenvpnProcessBasedConnectionFactory {
	return &OpenvpnProcessBasedConnectionFactory{
		mysteriumAPIClient:    mysteriumAPIClient,
		openvpnBinary:         openvpnBinary,
		configDirectory:       configDirectory,
		runtimeDirectory:      runtimeDirectory,
		statsKeeper:           statsKeeper,
		originalLocationCache: originalLocationCache,
		signerFactory:         signerFactory,
	}
}

func (op *OpenvpnProcessBasedConnectionFactory) getClientConfig(config json.RawMessage) (*ClientConfig, error) {
	var receivedConfig VPNConfig
	err := json.Unmarshal(config, &receivedConfig)
	if err != nil {
		return nil, err
	}

	vpnClientConfig, err := NewClientConfigFromSession(&receivedConfig, op.configDirectory, op.runtimeDirectory)
	return vpnClientConfig, err
}

func (op *OpenvpnProcessBasedConnectionFactory) newAuthMiddleware(sessionID session.ID, signer identity.Signer) management.Middleware {
	credentialsProvider := openvpn_session.SignatureCredentialsProvider(sessionID, signer)
	return auth.NewMiddleware(credentialsProvider)
}

func (op *OpenvpnProcessBasedConnectionFactory) newBytecountMiddleware() management.Middleware {
	statsSaver := bytescount.NewSessionStatsSaver(op.statsKeeper)
	return openvpn_bytescount.NewMiddleware(statsSaver, 1*time.Second)
}

func (op *OpenvpnProcessBasedConnectionFactory) newStateMiddleware(signer identity.Signer, connectionOptions connection.ConnectOptions, stateChannel connection.StateChannel) management.Middleware {
	statsSender := stats.NewRemoteStatsSender(
		op.statsKeeper,
		op.mysteriumAPIClient,
		connectionOptions.SessionID,
		connectionOptions.ProviderID,
		signer,
		op.originalLocationCache.Get().Country,
		time.Minute,
	)
	stateCallback := GetStateCallback(stateChannel, op.statsKeeper)
	return state.NewMiddleware(stateCallback, statsSender.StateHandler)
}

func (op *OpenvpnProcessBasedConnectionFactory) configureConnection(connectionOptions connection.ConnectOptions, stateChannel connection.StateChannel) (openvpn.Process, error) {
	vpnClientConfig, err := op.getClientConfig(connectionOptions.Config)
	if err != nil {
		return nil, err
	}

	signer := op.signerFactory(connectionOptions.ConsumerID)

	stateMiddleware := op.newStateMiddleware(signer, connectionOptions, stateChannel)
	authMiddleware := op.newAuthMiddleware(connectionOptions.SessionID, signer)
	byteCountMiddleware := op.newBytecountMiddleware()

	return NewClient(
		op.openvpnBinary,
		vpnClientConfig,
		stateMiddleware,
		byteCountMiddleware,
		authMiddleware,
	), nil
}

// CreateConnection implements the connection.ConnectionCreator interface
func (op *OpenvpnProcessBasedConnectionFactory) CreateConnection(connectionOptions connection.ConnectOptions, stateChannel connection.StateChannel) (connection.Connection, error) {
	return op.configureConnection(connectionOptions, stateChannel)
}
