/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package nat

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
)

// NamedPortProvider contains information of the NAT traversal method.
type NamedPortProvider struct {
	Method   string
	Provider PortProvider
}

// PortProvider describes an method for provideing ports for the service.
type PortProvider interface {
	PreparePorts() (ports []int, release func(), start StartPorts, err error)
}

var traversalOptions map[string]func() PortProvider = map[string]func() PortProvider{
	"manual":       NewManualPortProvider,
	"upnp":         NewUPnPPortProvider,
	"holepunching": NewNATHolePunchingPortProvider,
}

// OrderedPortProviders returns a ordered list of the port providers.
func OrderedPortProviders() (list []NamedPortProvider) {
	methods := strings.Split(config.GetString(config.FlagTraversal), ",")

	for _, m := range methods {
		if t, ok := traversalOptions[m]; ok {
			list = append(list, NamedPortProvider{Method: m, Provider: t()})
		} else {
			log.Warn().Msgf("Unsupported traversal method %s, ignoring it", m)
		}
	}

	if len(list) == 0 {
		log.Warn().Msg("Failed to parse ordered list of traversal methods, falling back to default values")

		return []NamedPortProvider{
			{"manual", NewManualPortProvider()},
			{"upnp", NewUPnPPortProvider()},
			{"holepunching", NewNATHolePunchingPortProvider()},
		}
	}

	return list
}
