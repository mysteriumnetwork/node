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

const (
	// Global scope overrides session scope and is not affected by session scope calls
	Global Scope = "global"
	// Session scope block is applied before connection session begins and is removed when session ends
	Session Scope = "session"
	// internal state to mark that no blocks are in effect
	none Scope = ""
)

// DefaultTrackingBlocker traffic blocker bootstrapped for global calls
var DefaultTrackingBlocker TrafficBlocker = &noopTrafficBlocker{}

// TrafficBlocker interface neededs to be satisfied by any implementations which provide firewall capabilities, like iptables
type TrafficBlocker interface {
	Setup() error
	Teardown()
	BlockOutgoingTraffic(scope Scope, outboundIP string) (RemoveRule, error)
	AllowIPAccess(ip string) (RemoveRule, error)
	AllowURLAccess(rawURLs ...string) (RemoveRule, error)
}

// Scope type represents scope of blocking consumer traffic
type Scope string

// RemoveRule type defines function for removal of created rule
type RemoveRule func()

// BlockNonTunnelTraffic effectively disallows any outgoing traffic from consumer node with specified scope
func BlockNonTunnelTraffic(scope Scope, outboundIP string) (RemoveRule, error) {
	return DefaultTrackingBlocker.BlockOutgoingTraffic(scope, outboundIP)
}

// AllowURLAccess adds exception to blocked traffic for specified URL (host part is usually taken)
func AllowURLAccess(urls ...string) (RemoveRule, error) {
	return DefaultTrackingBlocker.AllowURLAccess(urls...)
}

// AllowIPAccess adds IP based exception to underlying blocker implementation
func AllowIPAccess(ip string) (RemoveRule, error) {
	return DefaultTrackingBlocker.AllowIPAccess(ip)
}

// Reset firewall state - usually called when cleanup is needed (during shutdown)
func Reset() {
	DefaultTrackingBlocker.Teardown()
}
