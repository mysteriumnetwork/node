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

import (
	"net"
	"net/url"
	"sync"
)

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

type trafficBlocker struct {
	lock             sync.Mutex
	vendor           BlockVendor
	trafficLockScope Scope
}

func (tb *trafficBlocker) SwitchVendor(blocker BlockVendor) {
	tb.lock.Lock()
	defer tb.lock.Unlock()
	tb.vendor = blocker
}

func (tb *trafficBlocker) BlockOutgoingTraffic(scope Scope) (RemoveRule, error) {
	tb.lock.Lock()
	defer tb.lock.Unlock()
	if tb.trafficLockScope == Global {
		// nothing can override global lock
		return func() {}, nil
	}
	tb.trafficLockScope = scope
	return tb.vendor.BlockOutgoingTraffic()
}

func (tb *trafficBlocker) AllowIPAccess(ip string) (RemoveRule, error) {
	return tb.vendor.AllowIPAccess(ip)
}

func (tb *trafficBlocker) AllowURLAccess(rawURLs ...string) (RemoveRule, error) {
	var ruleRemovers []func()
	removeAll := func() {
		for _, ruleRemover := range ruleRemovers {
			ruleRemover()
		}
	}
	for _, rawURL := range rawURLs {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			removeAll()
			return nil, err
		}

		ips, err := net.LookupIP(parsed.Hostname())
		if err != nil {
			removeAll()
			return nil, err
		}

		for _, ip := range ips {
			remover, err := tb.AllowIPAccess(ip.String())
			if err != nil {
				removeAll()
				return nil, err
			}
			ruleRemovers = append(ruleRemovers, remover)
		}
	}
	return removeAll, nil

}

var currentBlocker = &trafficBlocker{
	vendor:           NoopVendor{LogPrefix: "[Noop firewall] "},
	trafficLockScope: none,
}

// Configure firewall with specified actual BlockVendor implementation
func Configure(blocker BlockVendor) {
	currentBlocker.SwitchVendor(blocker)
}

// BlockNonTunnelTraffic effectively disallows any outgoing traffic from consumer node with specified scope
func BlockNonTunnelTraffic(scope Scope) (RemoveRule, error) {
	return currentBlocker.BlockOutgoingTraffic(scope)
}

// AllowURLAccess adds exception to blocked traffic for specified URL (host part is usually taken)
func AllowURLAccess(rawURLs ...string) (RemoveRule, error) {
	return currentBlocker.AllowURLAccess(rawURLs...)
}

// AllowIPAccess adds IP based exception to underlying blocker implementation
func AllowIPAccess(ip string) (RemoveRule, error) {
	return currentBlocker.AllowIPAccess(ip)
}

// BlockVendor interface neededs to be satisfied by any implementations which provide firewall capabilities, like iptables
type BlockVendor interface {
	BlockOutgoingTraffic() (RemoveRule, error)
	AllowIPAccess(ip string) (RemoveRule, error)
	Reset()
}
