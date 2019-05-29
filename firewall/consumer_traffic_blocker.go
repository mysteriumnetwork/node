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

// RemoveRule type defines function for removal of created rule
type RemoveRule func()

// Scope type represents scope of blocking consumer traffic
type Scope string

const (
	// Global scope overrides session scope and is not affected by session scope calls
	Global Scope = "global"
	// Session scope block is applied before connection session begins and is removed when session ends
	Session Scope = "session"
	// internal state to mark that no blocks are in effect
	none Scope = "none"
)

var currentBlocker Blocker = NoopBlocker{
	LogPrefix: "[Noop firewall] ",
}

// Configure firewall with specified actual Blocker implementation
func Configure(blocker Blocker) {
	currentBlocker = blocker
}

// BlockNonTunnelTraffic effectively disallows any outgoing traffic from consumer node with specified scope
func BlockNonTunnelTraffic(scope Scope) (RemoveRule, error) {
	return currentBlocker.BlockNonTunnelTraffic(scope)
}

// AllowURLAccess adds exception to blocked traffic for specified URL (host part is usually taken)
func AllowURLAccess(url string) (RemoveRule, error) {
	return currentBlocker.AllowURLAccess(url)
}

// AllowIPAccess adds IP based exception to underlying blocker implementation
func AllowIPAccess(ip string) (RemoveRule, error) {
	return currentBlocker.AllowIPAccess(ip)
}

// Blocker interface neededs to be satisfied by any blocker implementations
type Blocker interface {
	BlockNonTunnelTraffic(scope Scope) (RemoveRule, error)
	AllowURLAccess(url string) (RemoveRule, error)
	AllowIPAccess(ip string) (RemoveRule, error)
}
