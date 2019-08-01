//+build !android

/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package vnd

import (
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall/iptables"
)

// SetupVendor initializes linux specific firewall vendor
func SetupVendor() (*iptables.Iptables, error) {
	resolver := ip.NewResolver("0.0.0.0", "")
	ip, err := resolver.GetOutboundIPAsString()
	if err != nil {
		return nil, err
	}
	iptables := iptables.New(ip)
	if err := iptables.Setup(); err != nil {
		return nil, err
	}
	return iptables, nil
}
