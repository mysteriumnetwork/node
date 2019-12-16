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

package nat

import (
	"net"
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"github.com/rs/zerolog/log"
)

func protectedNetworks() (nets []*net.IPNet) {
	cfg := config.GetString(config.FlagFirewallProtectedNetworks)
	if cfg == "" {
		return nil
	}
	for _, s := range strings.Split(cfg, ",") {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			log.Error().Err(err).Msg("Could not parse protected network string")
			continue
		}
		nets = append(nets, ipNet)
	}
	return nets
}
