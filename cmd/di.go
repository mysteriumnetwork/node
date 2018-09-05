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
	"path/filepath"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
)

// Dependencies is DI container for top level components which is reusedin several places
type Dependencies struct {
	NodeOptions node.Options
	Node        *node.Node

	IPResolver       ip.Resolver
	LocationResolver location.Resolver

	ServiceManager *service.Manager
}

// Bootstrap initiates all container dependencies
func (di *Dependencies) Bootstrap(nodeOptions node.Options) {
	di.bootstrapLocation(nodeOptions.Location, nodeOptions.Directories.Config)
	di.bootstrapNode(nodeOptions)
}

func (di *Dependencies) bootstrapNode(nodeOptions node.Options) {
	di.NodeOptions = nodeOptions
	di.Node = node.NewNode(
		nodeOptions,
		di.IPResolver,
		di.LocationResolver,
	)
}

// BootstrapServiceManager initiates ServiceManager dependency
func (di *Dependencies) BootstrapServiceManager(nodeOptions node.Options, serviceOptions service.Options) {
	di.ServiceManager = service.NewManager(
		nodeOptions,
		serviceOptions,
		di.IPResolver,
		di.LocationResolver,
	)
}

func (di *Dependencies) bootstrapLocation(options node.LocationOptions, configDirectory string) {
	di.IPResolver = ip.NewResolver(options.IpifyUrl)

	switch {
	case options.Country != "":
		di.LocationResolver = location.NewResolverFake(options.Country)
	default:
		di.LocationResolver = location.NewResolver(filepath.Join(configDirectory, options.Database))
	}
}
