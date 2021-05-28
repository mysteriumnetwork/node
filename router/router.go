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

package router

import (
	"net"
	"net/url"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/utils/netutil"
)

// AllowURLAccess adds exception to route traffic directly for specified URL (host part is usually taken).
func AllowURLAccess(urls ...string) error {
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			log.Info().Err(err).Msgf("Failed to parse URL: %s", u)
		}

		addresses, err := net.LookupHost(parsed.Hostname())
		if err != nil {
			log.Info().Err(err).Msgf("Excluding URL from the routes: %s", parsed.Hostname())
		}

		for _, a := range addresses {
			ipv4 := net.ParseIP(a)
			err := netutil.ExcludeRoute(ipv4)
			log.Info().Err(err).Msgf("Excluding URL address from the routes: %s -> %s", u, ipv4)
		}

	}
	return nil
}

// AllowIPAccess adds IP based exception to route traffic directly.
func AllowIPAccess(ip string) error {
	ipv4 := net.ParseIP(ip)
	err := netutil.ExcludeRoute(ipv4)
	if err != nil {
		log.Info().Err(err).Msgf("Excluding IP address from the routes: %s", ipv4)
	}

	return nil
}
