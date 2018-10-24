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
	"github.com/mysteriumnetwork/node/core/node"
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
		node.OptionsDirectory{
			dataDir,
			filepath.Join(dataDir, "db"),
			filepath.Join(dataDir, "keystore"),
			// TODO Embbed all config file to released artifacts
			filepath.Join(currentDir, "config"),
			// TODO Where to save runtime data
			currentDir,
		},

		"127.0.0.1",
		4050,

		openvpn_core.NodeOptions{},
		node.OptionsLocation{
			"https://api.ipify.org/",
			// TODO Embbed location DB to released artifacts
			"",
			"LT",
		},
		node.OptionsNetwork{Testnet: true},
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		log.Info("Shutdown Mysterium Node")
		di.Shutdown()
	}()

	proposals, err := di.MysteriumClient.FindProposals("0xd961ebabbdc17b7f82a18ef4f575d9e06f5a412d")
	if err != nil {
		panic(err)
	}
	if len(proposals) < 1 {
		panic("No FI server found")
	}

	log.Infof("Proposals: %#v", proposals)

	// TODO Return node startup/runtime errors to mobile
	err = di.Node.Wait()
	if err != nil {
		panic(err)
	}
}
