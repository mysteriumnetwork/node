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

package p2p

import (
	"net"
	"strings"

	"github.com/rs/zerolog/log"
)

func defaultInterfaceAddress() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to get list of interfaces")
		return ""
	}

	for _, i := range ifaces {
		if ip := upAndRunning(i, "wlan"); ip != "" {
			return ip
		}
	}

	for _, i := range ifaces {
		if ip := upAndRunning(i, "rmnet"); ip != "" {
			return ip
		}
	}

	return ""
}

func upAndRunning(iface net.Interface, prefix string) string {
	if !strings.HasPrefix(iface.Name, prefix) {
		return ""
	}

	if !strings.Contains(iface.Flags.String(), "up") {
		return ""
	}

	i, err := net.InterfaceByName(iface.Name)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to get interface by name: %s", iface.Name)
		return ""
	}

	addrs, err := i.Addrs()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to get interface addresses: %s", iface.Name)
		return ""
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to get interface addresses: %s", iface.Name)
			return ""
		}

		if ip.To4() == nil {
			return ""
		}

		return ip.String()
	}
	return ""
}
