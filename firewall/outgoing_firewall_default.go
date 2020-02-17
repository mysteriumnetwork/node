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
	// Global scope overrides session scope and is not affected by session scope calls.
	Global Scope = "global"
	// Session scope block is applied before connection session begins and is removed when session ends.
	Session Scope = "session"
	// internal state to mark that no blocks are in effect.
	none Scope = ""
)

// DefaultOutgoingFirewall outgoing traffic firewall bootstrapped for global calls.
var DefaultOutgoingFirewall OutgoingTrafficFirewall = &outgoingFirewallNoop{}

// OutgoingTrafficFirewall defines consumer side firewall a.k.a. kill switch.
// Purpose is to detect traffic which leaves machine and reject it,
// because during established VPN connection it is expected to leave through tunnel device only.
type OutgoingTrafficFirewall interface {
	Setup() error
	Teardown()
	BlockOutgoingTraffic(scope Scope, outboundIP string) (OutgoingRuleRemove, error)
	AllowIPAccess(ip string) (OutgoingRuleRemove, error)
	AllowURLAccess(rawURLs ...string) (OutgoingRuleRemove, error)
}

// Scope type represents scope of blocking consumer traffic.
type Scope string

// OutgoingRuleRemove type defines function for removal of created rule.
type OutgoingRuleRemove func()

// BlockNonTunnelTraffic effectively disallows any outgoing traffic from consumer node with specified scope.
func BlockNonTunnelTraffic(scope Scope, outboundIP string) (OutgoingRuleRemove, error) {
	return DefaultOutgoingFirewall.BlockOutgoingTraffic(scope, outboundIP)
}

// AllowURLAccess adds exception to blocked traffic for specified URL (host part is usually taken).
func AllowURLAccess(urls ...string) (OutgoingRuleRemove, error) {
	return DefaultOutgoingFirewall.AllowURLAccess(urls...)
}

// AllowIPAccess adds IP based exception.
func AllowIPAccess(ip string) (OutgoingRuleRemove, error) {
	return DefaultOutgoingFirewall.AllowIPAccess(ip)
}

// Reset firewall state - usually called when cleanup is needed (during shutdown).
func Reset() {
	DefaultOutgoingFirewall.Teardown()
}
