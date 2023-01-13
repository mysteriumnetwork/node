//go:build (ios || android) && !mobile_provider

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

package cmd

import (
	"net"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/tequilapi"
	uinoop "github.com/mysteriumnetwork/node/ui/noop"
	"github.com/mysteriumnetwork/node/ui/versionmanager"
)

func (di *Dependencies) bootstrapTequilapi(_ node.Options, _ net.Listener) (tequilapi.APIServer, error) {
	return tequilapi.NewNoopAPIServer(), nil
}

func (di *Dependencies) bootstrapUIServer(_ node.Options) (err error) {
	di.UIServer = uinoop.NewServer()
	return nil
}

func (di *Dependencies) bootstrapNodeUIVersionConfig(_ node.Options) error {
	noopCfg, _ := versionmanager.NewNoOpVersionConfig()
	di.uiVersionConfig = noopCfg
	return nil
}
