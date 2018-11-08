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
	"path/filepath"

	session2 "github.com/mysteriumnetwork/node/services/openvpn/session"

	"github.com/mysteriumnetwork/node/services/openvpn"

	"github.com/mysteriumnetwork/go-openvpn/openvpn3"

	log "github.com/cihub/seelog"
	"github.com/mitchellh/go-homedir"
	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
)

// MobileNode represents node object tuned for mobile devices
type MobileNode struct {
	di       cmd.Dependencies
	identity identity.Identity
}

// MobileNetworkOptions alias for node.OptionsNetwork to be visible from mobile framework
type MobileNetworkOptions node.OptionsNetwork

type Openvpn3TunnelSetup openvpn3.TunnelSetup

// NewNode function creates new Node
func NewNode(appPath string, optionsNetwork *MobileNetworkOptions, tunnelSetup Openvpn3TunnelSetup) (*MobileNode, error) {
	var di cmd.Dependencies

	var dataDir, currentDir string
	if appPath == "" {
		currentDir, err := homedir.Dir()
		if err != nil {
			panic(err)
		}
		dataDir = filepath.Join(currentDir, ".mysterium")
	} else {
		dataDir = filepath.Join(appPath, ".mysterium")
		currentDir = appPath
	}

	err := di.Bootstrap(node.Options{
		Directories: node.OptionsDirectory{
			Data:     dataDir,
			Storage:  filepath.Join(dataDir, "db"),
			Keystore: filepath.Join(dataDir, "keystore"),
			// TODO Where to save runtime data
			Runtime: currentDir,
		},

		TequilapiAddress: "127.0.0.1",
		TequilapiPort:    4050,

		// TODO Make Openvpn pluggable connection optional
		Openvpn: noOpenvpnYet{},

		Location: node.OptionsLocation{
			IpifyUrl: "https://api.ipify.org/",
			Country:  "LT",
		},

		OptionsNetwork: node.OptionsNetwork(*optionsNetwork),
	})
	if err != nil {
		return nil, err
	}

	di.ConnectionRegistry.Register("openvpn", func(options connection.ConnectOptions, channel connection.StateChannel) (connection.Connection, error) {

		vpnClientConfig, err := openvpn.NewClientConfigFromSession(options.SessionConfig, "", "")
		if err != nil {
			return nil, err
		}

		profileContent, err := vpnClientConfig.ToConfigFileContent()
		if err != nil {
			return nil, err
		}

		config := openvpn3.NewConfig(profileContent)
		config.GuiVersion = "govpn 0.1"
		config.CompressionMode = "asym"

		signer := di.SignerFactory(options.ConsumerID)

		username, password, err := session2.SignatureCredentialsProvider(options.SessionID, signer)()
		if err != nil {
			return nil, err
		}

		credentials := openvpn3.UserCredentials{
			Username: username,
			Password: password,
		}

		session := openvpn3.NewMobileSession(config, credentials, channelToCallbacks(channel, di.StatsKeeper), tunnelSetup)

		return &sessionWrapper{
			session: session,
		}, nil
	})

	return &MobileNode{di: di}, nil
}

// DefaultNetworkOptions returns default network options to connect with
func DefaultNetworkOptions() *MobileNetworkOptions {
	return &MobileNetworkOptions{
		Testnet:                 true,
		ExperimentIdentityCheck: true,
		DiscoveryAPIAddress:     metadata.TestnetDefinition.DiscoveryAPIAddress,
		BrokerAddress:           metadata.TestnetDefinition.BrokerAddress,
		EtherClientRPC:          metadata.TestnetDefinition.EtherClientRPC,
		EtherPaymentsAddress:    metadata.DefaultNetwork.PaymentsContractAddress.String(),
	}
}

// TestConnectFlow checks whenever connection can be successfully established
func (mobNode *MobileNode) TestConnectFlow(providerAddress string) error {
	providerId := identity.FromAddress(providerAddress)
	log.Infof("Connecting to provider: %#v", providerId)
	err := mobNode.di.ConnectionManager.Connect(mobNode.identity, providerId, connection.ConnectParams{})
	if err != nil {
		return err
	}

	connectionStatus := mobNode.di.ConnectionManager.Status()
	log.Infof("Connection status: %#v", connectionStatus)

	return mobNode.di.ConnectionManager.Disconnect()
}

// TestConnectFlow checks whenever connection can be successfully established
func (mobNode *MobileNode) CreateIdentity() (string, error) {
	consumers := mobNode.di.IdentityManager.GetIdentities()
	var consumerID identity.Identity
	if len(consumers) < 1 {
		created, err := mobNode.di.IdentityManager.CreateNewIdentity("")
		if err != nil {
			return "", err
		}
		consumerID = created
	} else {
		consumerID = consumers[0]
	}

	log.Infof("Unlocking consumer: %#v", consumerID)
	err := mobNode.di.IdentityManager.Unlock(consumerID.Address, "")
	if err != nil {
		return "", err
	}
	mobNode.identity = consumerID
	return consumerID.Address, nil
}

// Shutdown function stops running mobile node
func (mobNode *MobileNode) Shutdown() error {
	return mobNode.di.Node.Kill()
}

// WaitUntilDies function returns when node stops
func (mobNode *MobileNode) WaitUntilDies() error {
	return mobNode.di.Node.Wait()
}

type noOpenvpnYet struct {
}

func (noOpenvpnYet) Check() error {
	return nil
}

// BinaryPath returns noop binary path
func (noOpenvpnYet) BinaryPath() string {
	return "no openvpn binary available on mobile"
}

var _ node.Openvpn = noOpenvpnYet{}
