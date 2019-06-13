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

package firewall

var trackingBlocker = newTrackingBlocker()

// Configure blocker with specified actual Vendor implementation
func Configure(vendor Vendor) {
	trackingBlocker.SwitchVendor(vendor)
}

// BlockNonTunnelTraffic effectively disallows any outgoing traffic from consumer node with specified scope
func BlockNonTunnelTraffic(scope Scope) (RemoveRule, error) {
	return trackingBlocker.BlockOutgoingTraffic(scope)
}

// AllowURLAccess adds exception to blocked traffic for specified URL (host part is usually taken)
func AllowURLAccess(urls ...string) (RemoveRule, error) {
	return trackingBlocker.AllowURLAccess(urls...)
}

// AllowIPAccess adds IP based exception to underlying blocker implementation
func AllowIPAccess(ip string) (RemoveRule, error) {
	return trackingBlocker.AllowIPAccess(ip)
}

// Reset firewall state - usually called when cleanup is needed (during shutdown)
func Reset() {
	trackingBlocker.vendor.Reset()
}

// Vendor interface neededs to be satisfied by any implementations which provide firewall capabilities, like iptables
type Vendor interface {
	BlockOutgoingTraffic() (RemoveRule, error)
	AllowIPAccess(ip string) (RemoveRule, error)
	Reset()
}
