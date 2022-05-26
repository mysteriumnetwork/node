/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package ip

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CachedResolver resolves IP and caches for some duration.
type CachedResolver struct {
	resolver      Resolver
	cacheDuration time.Duration

	outboundIP         string
	outboundIPLock     sync.Mutex
	outboundIPCachedAt time.Time

	publicIP         string
	publicIPLock     sync.Mutex
	publicIPCachedAt time.Time
}

// NewCachedResolver creates ip resolver with cache duration.
func NewCachedResolver(resolver Resolver, cacheDuration time.Duration) *CachedResolver {
	return &CachedResolver{
		resolver:      resolver,
		cacheDuration: cacheDuration,
	}
}

// GetOutboundIP returns current outbound IP as string for current system.
func (r *CachedResolver) GetOutboundIP() (string, error) {
	r.outboundIPLock.Lock()
	defer r.outboundIPLock.Unlock()

	if r.outboundIPCachedAt.Add(r.cacheDuration).After(time.Now()) && r.outboundIP != "" {
		log.Debug().Msgf("Found cached outbound IP")
		return r.outboundIP, nil
	}

	log.Debug().Msg("Outbound IP cache is empty, fetching IP")
	outboundIP, err := r.resolver.GetOutboundIP()
	if err != nil {
		return "", err
	}
	r.outboundIPCachedAt = time.Now()
	r.outboundIP = outboundIP
	return r.outboundIP, nil
}

// GetPublicIP returns current public IP.
func (r *CachedResolver) GetPublicIP() (string, error) {
	r.publicIPLock.Lock()
	defer r.publicIPLock.Unlock()

	if r.publicIPCachedAt.Add(r.cacheDuration).After(time.Now()) && r.publicIP != "" {
		log.Debug().Msgf("Found cached public IP")
		return r.publicIP, nil
	}

	log.Debug().Msg("Public IP cache is empty, fetching IP")
	publicIP, err := r.resolver.GetPublicIP()
	if err != nil {
		return "", err
	}
	r.publicIPCachedAt = time.Now()
	r.publicIP = publicIP
	return r.publicIP, nil
}

// GetProxyIP returns proxy public IP.
func (r *CachedResolver) GetProxyIP(proxyPort int) (string, error) {
	publicIP, err := r.resolver.GetProxyIP(proxyPort)
	if err != nil {
		return "", err
	}

	return publicIP, nil
}

// ClearCache clears ip cache.
func (r *CachedResolver) ClearCache() {
	log.Debug().Msg("Clearing ip resolver cache")

	r.outboundIPLock.Lock()
	r.outboundIP = ""
	r.outboundIPCachedAt = time.Time{}
	r.outboundIPLock.Unlock()

	r.publicIPLock.Lock()
	r.publicIP = ""
	r.publicIPCachedAt = time.Time{}
	r.publicIPLock.Unlock()
}
