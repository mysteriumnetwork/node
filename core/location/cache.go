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

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
)

// Cache allows us to cache location resolution
type Cache struct {
	lastFetched      time.Time
	locationDetector Resolver
	location         locationstate.Location
	origin           locationstate.Location
	expiry           time.Duration
	pub              publisher
	lock             sync.Mutex
}

type publisher interface {
	Publish(topic string, data interface{})
}

// LocUpdateEvent is the event type used to sending or receiving event updates
const LocUpdateEvent string = "location-update-event"

// NewCache returns a new instance of location cache
func NewCache(resolver Resolver, pub publisher, expiry time.Duration) *Cache {
	return &Cache{
		locationDetector: resolver,
		expiry:           expiry,
		pub:              pub,
	}
}

func (c *Cache) needsRefresh() bool {
	return c.lastFetched.IsZero() || c.lastFetched.After(time.Now().Add(-c.expiry))
}

func (c *Cache) fetchAndSave() (locationstate.Location, error) {
	loc, err := c.locationDetector.DetectLocation()

	// on successful fetch save the values for further use
	if err == nil {
		ip := loc.IP
		// avoid printing IP address in logs
		loc.IP = ""
		c.pub.Publish(LocUpdateEvent, loc)
		loc.IP = ip
		c.location = loc
		c.lastFetched = time.Now()
	}
	return loc, err
}

// GetOrigin returns the origin for the user - a location that's not modified by starting services.
func (c *Cache) GetOrigin() locationstate.Location {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.origin
}

// DetectLocation returns location from cache, or fetches it if needed
func (c *Cache) DetectLocation() (locationstate.Location, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.needsRefresh() {
		return c.location, nil
	}
	return c.fetchAndSave()
}

// DetectProxyLocation returns the proxy location.
func (c *Cache) DetectProxyLocation(proxyPort int) (locationstate.Location, error) {
	return c.locationDetector.DetectProxyLocation(proxyPort)
}

// HandleConnectionEvent handles connection state change and fetches the location info accordingly.
// On the consumer side, we'll need to re-fetch the location once the user is connected or disconnected from a service.
func (c *Cache) HandleConnectionEvent(se connectionstate.AppEventConnectionState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if se.State != connectionstate.Connected && se.State != connectionstate.NotConnected {
		return
	}

	_, err := c.fetchAndSave()
	if err != nil {
		log.Error().Err(err).Msg("Location update failed")
		// reset time so a fetch is tried the next time a get is called
		c.lastFetched = time.Time{}
	}
}

// HandleNodeEvent handles node state change and fetches the location info accordingly.
func (c *Cache) HandleNodeEvent(se nodevent.Payload) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if se.Status != nodevent.StatusStarted {
		return
	}

	var err error
	c.origin, err = c.locationDetector.DetectLocation()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to detect original location")
	} else {
		log.Debug().Msgf("original location detected: %s (%s)", c.origin.Country, c.origin.IPType)
	}
}
