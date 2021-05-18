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

package requests

import "sync"

var defaultResolveCache = NewResolverCache()

type resolverCache struct {
	mu    sync.Mutex
	cache map[string][]string
}

// NewResolverCache caches resolver responses.
func NewResolverCache() *resolverCache {
	return &resolverCache{
		cache: make(map[string][]string),
	}
}

// CacheDNSRecord add a cached record for the provided name.
func CacheDNSRecord(name string, addrs []string) {
	defaultResolveCache.Add(name, addrs)
}

// FetchDNSFromCache returns a cached addressed for the provided name.
func FetchDNSFromCache(name string) (addrs []string) {
	return defaultResolveCache.Fetch(name)
}

func (rc *resolverCache) Fetch(name string) []string {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	return rc.cache[name]
}

func (rc *resolverCache) Add(name string, addrs []string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache[name] = addrs
}
