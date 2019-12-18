/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

import "net"

// NATService routes internet traffic through provider and
// sets up firewall rules for security
type NATService interface {
	Enable() error
	Setup(opts Options) (rules []interface{}, err error)
	Del(rules []interface{}) error
	Disable() error
}

// Options params to setup firewall/NAT rules.
type Options struct {
	VPNNetwork        net.IPNet
	ProviderExtIP     net.IP
	EnableDNSRedirect bool
	DNSIP             net.IP
	DNSPort           int
}
