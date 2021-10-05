/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package netutil

import (
	"net"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogNetworkStats logs network information to the Trace log level.
var LogNetworkStats = defaultLogNetworkStats

// AddDefaultRoute adds default VPN tunnel route.
func AddDefaultRoute(iface string) error {
	return addDefaultRoute(iface)
}

// AssignIP assigns subnet to given interface.
func AssignIP(iface string, subnet net.IPNet) error {
	return assignIP(iface, subnet)
}

func defaultLogNetworkStats() {
	if log.Logger.GetLevel() != zerolog.TraceLevel {
		return
	}

	logNetworkStats()
}

func logOutputToTrace(out []byte, err error, args ...string) {
	logSkipFrame := log.With().CallerWithSkipFrameCount(3).Logger()

	if err != nil {
		(&logSkipFrame).Trace().Msgf("Failed to get %s error: %v", strings.Join(args, " "), err)
	} else {
		(&logSkipFrame).Trace().Msgf("%q output:\n%s", strings.Join(args, " "), out)
	}
}
