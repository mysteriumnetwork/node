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

package location

import (
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/connection"
)

type cache struct {
	locationDetector Resolver
	location         Location
}

// NewLocationCache constructs Cache
func NewLocationCache(locationDetector Resolver) Cache {
	return &cache{
		locationDetector: locationDetector,
	}
}

// Gets location from cache
func (lc *cache) Get() Location {
	return lc.location
}

// Stores location to cache
func (lc *cache) RefreshAndGet() (Location, error) {
	location, err := lc.locationDetector.DetectLocation()
	lc.location = location
	return lc.location, err
}

const locationCacheLogPrefix = "[location-cache]"

type ProperCache struct {
	lastFetched      time.Time
	locationDetector Resolver
	location         Location
	expiry           time.Duration
	lock             sync.Mutex
}

func (c *ProperCache) needsRefresh() bool {
	return c.lastFetched.IsZero() || c.lastFetched.After(time.Now().Add(-c.expiry))
}

func (c *ProperCache) fetchAndSave() (Location, error) {
	loc, err := c.locationDetector.DetectLocation()

	// on successful fetch save the values for further use
	if err == nil {
		c.location = loc
		c.lastFetched = time.Now()
	}

	return loc, err
}

// Get returns location from cache, or fetches it if needed
func (c *ProperCache) Get() (Location, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.needsRefresh() {
		return c.location, nil
	}

	return c.fetchAndSave()
}

// HandleConnectionEvent handles connection state change and fetches the location info accordingly.
// On the consumer side, we'll need to re-fetch the location once the user is connected or disconnected from a service.
func (c *ProperCache) HandleConnectionEvent(se connection.StateEvent) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if se.State != connection.Connected && se.State != connection.NotConnected {
		return
	}

	_, err := c.fetchAndSave()
	if err != nil {
		log.Error(locationCacheLogPrefix, "location update failed", err)
		// reset time so a fetch is tried the next time a get is called
		c.lastFetched = time.Time{}
	} else {
		log.Trace(locationCacheLogPrefix, "location update succeeded")
	}
}
