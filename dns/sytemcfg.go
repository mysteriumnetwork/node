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

package dns

import (
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// configuration returns the system DNS configuration.
func configuration() (*dns.ClientConfig, error) {
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	return config, errors.Wrap(err, "error loading DNS config")
}

// ConfiguredServers returns DNS server IPs from the system DNS configuration.
func ConfiguredServers() ([]string, error) {
	config, err := configuration()
	if err != nil {
		return nil, err
	}
	return config.Servers, nil
}
