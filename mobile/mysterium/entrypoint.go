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

	log "github.com/cihub/seelog"
	"github.com/mitchellh/go-homedir"
	openvpn_core "github.com/mysteriumnetwork/go-openvpn/openvpn/core"
	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
)

// NewNode function creates new Node
func NewNode() {
	var di cmd.Dependencies

	userHomeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dataDir := filepath.Join(userHomeDir, ".mysterium")
	currentDir := userHomeDir

	err = di.Bootstrap(node.Options{
		Directories: node.OptionsDirectory{
			Data:     dataDir,
			Storage:  filepath.Join(dataDir, "db"),
			Keystore: filepath.Join(dataDir, "keystore"),
			// TODO Embbed all config file to released artifacts
			Config: filepath.Join(currentDir, "config"),
			// TODO Where to save runtime data
			Runtime: currentDir,
		},

		TequilapiAddress: "127.0.0.1",
		TequilapiPort:    4050,

		// TODO Make Openvpn pluggable connection optional
		Openvpn: openvpn_core.NodeOptions{},

		Location: node.OptionsLocation{
			IpifyUrl: "https://api.ipify.org/",
			Country:  "LT",
		},
		OptionsNetwork: node.OptionsNetwork{
			Testnet:              true,
			DiscoveryAPIAddress:  metadata.TestnetDefinition.DiscoveryAPIAddress,
			BrokerAddress:        metadata.TestnetDefinition.BrokerAddress,
			EtherClientRPC:       metadata.TestnetDefinition.EtherClientRPC,
			EtherPaymentsAddress: metadata.DefaultNetwork.PaymentsContractAddress.String(),
		},
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		log.Info("Shutdown Mysterium Node")
		di.Shutdown()
	}()

	// TODO Mock transport until encrypted tunnel is ready in mobile
	di.ConnectionRegistry.Register("openvpn", service_noop.NewConnectionCreator())

	// TODO Remove later, this is for initial green path test only
	//testConnectFlow(&di, identity.FromAddress("0xd961ebabbdc17b7f82a18ef4f575d9e06f5a412d"))

	// TODO Return node startup/runtime errors to mobile
	err = di.Node.Wait()
	if err != nil {
		panic(err)
	}
}

func testConnectFlow(di *cmd.Dependencies, providerID identity.Identity) {
	consumers := di.IdentityManager.GetIdentities()
	if len(consumers) < 1 {
		panic("No identity found")
	}
	consumerID := consumers[0]

	log.Infof("Unlocking consumer: %#v", consumerID)
	err := di.IdentityManager.Unlock(consumerID.Address, "")
	if err != nil {
		panic(err)
	}

	log.Infof("Connecting to provider: %#v", providerID)
	err = di.ConnectionManager.Connect(consumerID, providerID, connection.ConnectParams{})
	if err != nil {
		panic(err)
	}

	connectionStatus := di.ConnectionManager.Status()
	log.Infof("Connection status: %#v", connectionStatus)
	err = di.ConnectionManager.Disconnect()
	if err != nil {
		panic(err)
	}

}
