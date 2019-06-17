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
	none Scope = ""
)

type refCount struct {
	count int
	f     func()
}

type referenceTrackingBlocker struct {
	lock             sync.Mutex
	vendor           Vendor
	trafficLockScope Scope
	referenceTracker map[string]refCount
}

func newTrackingBlocker() *referenceTrackingBlocker {
	return &referenceTrackingBlocker{
		vendor:           NoopVendor{LogPrefix: "[Noop firewall]"},
		referenceTracker: make(map[string]refCount),
		trafficLockScope: none,
	}
}

func (tb *referenceTrackingBlocker) SwitchVendor(vendor Vendor) {
	tb.lock.Lock()
	defer tb.lock.Unlock()
	tb.vendor = vendor
}

func (tb *referenceTrackingBlocker) BlockOutgoingTraffic(scope Scope) (RemoveRule, error) {
	if tb.trafficLockScope == Global {
		// nothing can override global lock
		return func() {}, nil
	}
	tb.trafficLockScope = scope
	return tb.trackingReferenceCall("block-traffic", tb.vendor.BlockOutgoingTraffic)
}

func (tb *referenceTrackingBlocker) AllowIPAccess(ip string) (RemoveRule, error) {
	return tb.trackingReferenceCall("allow:"+ip, func() (rule RemoveRule, e error) {
		return tb.vendor.AllowIPAccess(ip)
	})
}

func (tb *referenceTrackingBlocker) AllowURLAccess(rawURLs ...string) (RemoveRule, error) {
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

		remover, err := tb.AllowIPAccess(parsed.Hostname())
		if err != nil {
			removeAll()
			return nil, err
		}
		ruleRemovers = append(ruleRemovers, remover)
	}
	return removeAll, nil

}

func (tb *referenceTrackingBlocker) trackingReferenceCall(ref string, actualCall func() (RemoveRule, error)) (RemoveRule, error) {
	tb.lock.Lock()
	defer tb.lock.Unlock()
	refCount := tb.referenceTracker[ref]
	if refCount.count == 0 {
		removeRule, err := actualCall()
		if err != nil {
			return nil, err
		}
		refCount.f = removeRule
	}
	refCount.count++
	tb.referenceTracker[ref] = refCount
	return tb.decreaseRefCall(ref), nil
}

func (tb *referenceTrackingBlocker) decreaseRefCall(ref string) RemoveRule {
	return func() {
		tb.lock.Lock()
		defer tb.lock.Unlock()
		refCount := tb.referenceTracker[ref]
		refCount.count--
		if refCount.count == 0 {
			refCount.f()
		}
		tb.referenceTracker[ref] = refCount
	}
}
