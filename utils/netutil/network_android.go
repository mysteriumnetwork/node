//go:build android

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

package netutil

import (
	"net"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

func assignIP(iface string, subnet net.IPNet) error {
	return nil
}

func excludeRoute(ip, gw net.IP) error {
	return nil
}

func deleteRoute(ip, gw string) error {
	return nil
}

func addDefaultRoute(iface string) error {
	return nil
}

func logNetworkStats() {
}

func ipv6Enabled() bool {
	out, err := cmdutil.ExecOutput("sysctl", "net.ipv6.conf.all.disable_ipv6")
	if err != nil {
		log.Error().Err(err).Msg("Failed to detect if IPv6 disabled on the host")
		return true
	}

	return strings.Contains(string(out), "net.ipv6.conf.all.disable_ipv6 = 0")
}
