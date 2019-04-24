/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package factory

import (
	"fmt"
	"path/filepath"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
)

func CreateResolver(options node.OptionsLocation, configDirectory string, ipResolver ip.Resolver) (
	location.Resolver,
	error,
) {
	manualResolver := location.NewStaticResolver(options.Country, options.City, options.NodeType, ipResolver)

	switch options.Type {
	case node.LocationTypeManual:
		return location.NewFallbackResolver([]location.Resolver{manualResolver}), nil

	case node.LocationTypeBuiltin:
		var builtinResolver location.Resolver
		builtinResolver, err := location.NewBuiltInResolver(ipResolver)
		if err != nil {
			log.Error("Failed to load builtin location resolver: ", err)
			builtinResolver = location.NewFailingResolver(err)
		}
		return location.NewFallbackResolver([]location.Resolver{builtinResolver, manualResolver}), nil

	case node.LocationTypeMMDB:
		var mmdbResolver location.Resolver
		mmdbResolver, err := location.NewExternalDBResolver(filepath.Join(configDirectory, options.Address), ipResolver)
		if err != nil {
			log.Error("Failed to load external db location resolver: ", err)
			mmdbResolver = location.NewFailingResolver(err)
		}
		return location.NewFallbackResolver([]location.Resolver{mmdbResolver, manualResolver}), nil

	case node.LocationTypeOracle:
		oracleResolver := location.NewOracleResolver(options.Address)
		return location.NewFallbackResolver([]location.Resolver{oracleResolver, manualResolver}), nil

	default:
		return nil, fmt.Errorf("unknown location detector type: %s", options.Type)
	}
}
