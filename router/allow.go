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
)

var defaultRouter = NewManager()

// ExcludeURL adds exception to route traffic directly for specified URL (host part is usually taken).
func ExcludeURL(urls ...string) error {
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to parse URL: %s", u)
			continue
		}

		addresses, err := net.LookupHost(parsed.Hostname())
		if err != nil {
			log.Error().Err(err).Msgf("Failed to exclude URL from routes: %s", parsed.Hostname())
			continue
		}

		for _, a := range addresses {
			ipv4 := net.ParseIP(a)
			err := defaultRouter.ExcludeIP(ipv4)
			log.Info().Err(err).Msgf("Excluding URL address from the routes: %s -> %s", u, ipv4)
		}
	}

	return nil
}

// ExcludeIP adds IP based exception to route traffic directly.
func ExcludeIP(ip net.IP) error {
	err := defaultRouter.ExcludeIP(ip)
	if err != nil {
		log.Info().Err(err).Msgf("Excluding IP address from the routes: %s", ip)
	}

	return nil
}

// Clean removes all previously added routing rules.
func Clean() error {
	err := defaultRouter.Clean()
	if err != nil {
		log.Info().Err(err).Msgf("Failed to clean")
	}

	return nil
}
