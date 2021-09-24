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

package domain

import (
	"strings"
)

// NormalizeHostname casts FQDN to canonical representation:
// no whitespace, no trailing dot, lower case.
func NormalizeHostname(hostname string) string {
	return strings.ToLower(
		strings.TrimRight(
			strings.TrimSpace(hostname),
			".",
		),
	)
}

// Whitelist is a set of domain names and suffixes for fast matching
type Whitelist struct {
	exactList  map[string]struct{}
	suffixList map[string]struct{}
}

// NewWhitelist creates Whitelist from list of domains and suffixes
func NewWhitelist(domainList []string) *Whitelist {
	exactList := make(map[string]struct{})
	suffixList := make(map[string]struct{})
	for _, domain := range domainList {
		domain = strings.TrimSpace(domain)
		normalized := NormalizeHostname(domain)
		if strings.Index(domain, ".") == 0 {
			// suffix pattern
			suffixList[strings.TrimLeft(normalized, ".")] = struct{}{}
		} else {
			// exact domain name
			exactList[normalized] = struct{}{}
		}
	}
	return &Whitelist{
		exactList:  exactList,
		suffixList: suffixList,
	}
}

// Match returns whether hostname is present in whitelist
func (l *Whitelist) Match(hostname string) bool {
	hostname = NormalizeHostname(hostname)

	// check for exact match
	if _, found := l.exactList[hostname]; found {
		return true
	}

	// handle special case of root suffix (".") added to whitelist
	if _, found := l.suffixList[""]; found && hostname != "" {
		return true
	}

	// check for suffix match
	for needle := strings.Split(hostname, ".")[1:]; len(needle) > 0; needle = needle[1:] {
		if _, found := l.suffixList[strings.Join(needle, ".")]; found {
			return true
		}
	}

	return false
}
