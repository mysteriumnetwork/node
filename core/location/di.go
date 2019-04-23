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

package location

import (
	"fmt"
	"path/filepath"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/ip"
)

// CreateLocationResolver creates a fallback resolver given the params
func CreateLocationResolver(ipResolver ip.Resolver, country, city, locatorType, nodeType, address, externalDb, configDirectory string) (Resolver, error) {
	if locatorType == "manual" {
		return NewStaticResolver(country, city, nodeType, ipResolver), nil
	}

	var builtin Resolver
	builtin, err := NewBuiltInResolver(ipResolver)
	if err != nil {
		log.Error(fallbackResolverLogPrefix, "Failed to load builtin location resolver: ", err)
		builtin = NewFailingResolver(err)
	}

	oracleResolver := NewOracleResolver(address)

	switch locatorType {
	case "builtin":
		return NewFallbackResolver([]Resolver{builtin, oracleResolver}), nil
	case "mmdb":
		var mmdb Resolver
		mmdb, err = NewExternalDBResolver(filepath.Join(configDirectory, externalDb), ipResolver)
		if err != nil {
			log.Error(fallbackResolverLogPrefix, "Failed to load external db location resolver: ", err)
			mmdb = NewFailingResolver(err)
		}
		return NewFallbackResolver([]Resolver{mmdb, oracleResolver, builtin}), nil
	case "", "oracle":
		return NewFallbackResolver([]Resolver{oracleResolver, builtin}), nil
	default:
		return nil, fmt.Errorf("unknown location detector type: %s", locatorType)
	}
}
