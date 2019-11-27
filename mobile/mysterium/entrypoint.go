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

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
)

// MobileNode represents node object tuned for mobile devices
type MobileNode struct {
	di cmd.Dependencies
}

// MobileNetworkOptions alias for node.OptionsNetwork to be visible from mobile framework
type MobileNetworkOptions node.OptionsNetwork

// MobileLogOptions alias for logconfig.LogOptions
type MobileLogOptions logconfig.LogOptions

// NewNode function creates new Node
func NewNode(appPath string, logOptions *MobileLogOptions, optionsNetwork *MobileNetworkOptions) (*MobileNode, error) {
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

	network := node.OptionsNetwork(*optionsNetwork)
	log := logconfig.LogOptions(*logOptions)

	err := di.Bootstrap(node.Options{
		LogOptions: log,
		Directories: node.OptionsDirectory{
			Data:     dataDir,
			Storage:  filepath.Join(dataDir, "db"),
			Keystore: filepath.Join(dataDir, "keystore"),
			Runtime:  currentDir,
		},

		TequilapiAddress: "127.0.0.1",
		TequilapiPort:    4050,

		Openvpn: embeddedLibCheck{},

		Keystore: node.OptionsKeystore{
			UseLightweight: true,
		},

		OptionsNetwork: network,
		Quality: node.OptionsQuality{
			Type:    node.QualityTypeMORQA,
			Address: "https://quality.mysterium.network/api/v1",
		},
		Discovery: node.OptionsDiscovery{
			Type:    node.DiscoveryTypeAPI,
			Address: network.MysteriumAPIAddress,
		},
		Location: node.OptionsLocation{
			IPDetectorURL: "https://api.ipify.org/?format=json",
			Type:          node.LocationTypeOracle,
			Address:       "https://testnet-location.mysterium.network/api/v1/location",
		},
	})
	if err != nil {
		return nil, err
	}

	return &MobileNode{di: di}, nil
}

// DefaultLogOptions default logging options for mobile
func DefaultLogOptions() *MobileLogOptions {
	return &MobileLogOptions{
		LogLevel: zerolog.DebugLevel,
	}
}

// DefaultNetworkOptions returns default network options to connect with
func DefaultNetworkOptions() *MobileNetworkOptions {
	return &MobileNetworkOptions{
		Testnet:               true,
		ExperimentNATPunching: true,
		MysteriumAPIAddress:   metadata.TestnetDefinition.MysteriumAPIAddress,
		BrokerAddress:         metadata.TestnetDefinition.BrokerAddress,
		EtherClientRPC:        metadata.TestnetDefinition.EtherClientRPC,
	}
}

// Shutdown function stops running mobile node
func (mobNode *MobileNode) Shutdown() error {
	return mobNode.di.Node.Kill()
}

// WaitUntilDies function returns when node stops
func (mobNode *MobileNode) WaitUntilDies() error {
	return mobNode.di.Node.Wait()
}

type embeddedLibCheck struct {
}

// Check always returns nil as embedded lib does not have any external failing deps
func (embeddedLibCheck) Check() error {
	return nil
}

// BinaryPath returns noop binary path
func (embeddedLibCheck) BinaryPath() string {
	return "mobile uses embedded openvpn lib"
}

// check if our struct satisfies Openvpn interface expected by node options
var _ node.Openvpn = embeddedLibCheck{}
