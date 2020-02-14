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

package mapping

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/mocks"
	"github.com/stretchr/testify/assert"
)

func TestMap_uPnP_Enabled(t *testing.T) {
	router := &mockRouter{uPnPEnabled: true}
	config := &Config{
		MapInterface:      router,
		MapUpdateInterval: 5 * time.Millisecond,
		MapLifetime:       10 * time.Millisecond,
	}
	portMapper := NewPortMapper(config, mocks.NewEventBus())

	release, ok := portMapper.Map("UDP", 51334, "Test")
	time.Sleep(config.MapUpdateInterval * 3)
	defer release()

	assert.True(t, ok)
	assert.Equal(t, mapping{
		protocol: "UDP",
		extport:  51334,
		intport:  51334,
		name:     "Test",
		lifetime: config.MapLifetime,
	}, router.addedMapping())
}

func TestMap_uPnP_Enabled_With_Permanent_Lease(t *testing.T) {
	router := &mockRouter{uPnPEnabled: true, permanentLease: true}
	config := &Config{
		MapInterface:      router,
		MapUpdateInterval: 5 * time.Millisecond,
		MapLifetime:       10 * time.Millisecond,
	}
	portMapper := NewPortMapper(config, mocks.NewEventBus())

	release, ok := portMapper.Map("UDP", 51334, "Test")
	time.Sleep(config.MapUpdateInterval * 3)
	defer release()

	assert.True(t, ok)
	assert.Equal(t, mapping{
		protocol: "UDP",
		extport:  51334,
		intport:  51334,
		name:     "Test",
		lifetime: 0,
	}, router.addedMapping())
}

func TestMap_uPnP_Disabled(t *testing.T) {
	router := &mockRouter{uPnPEnabled: false}
	config := &Config{
		MapInterface: router,
	}
	portMapper := NewPortMapper(config, mocks.NewEventBus())

	release, ok := portMapper.Map("UDP", 51334, "Test port mapping")

	assert.Nil(t, release)
	assert.False(t, ok)
	assert.Equal(t, mapping{}, router.addedMapping())
}

type mapping struct {
	protocol         string
	extport, intport int
	name             string
	lifetime         time.Duration
}

type mockRouter struct {
	sync.Mutex
	uPnPEnabled    bool
	permanentLease bool

	mapping mapping
}

func (m *mockRouter) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	m.Lock()
	defer m.Unlock()

	if !m.uPnPEnabled {
		return errors.New("uPnP not supported")
	}
	if m.permanentLease && lifetime > 0 {
		return errors.New("router supports permanent port lease only")
	}
	m.mapping = mapping{
		protocol: protocol,
		extport:  extport,
		intport:  intport,
		name:     name,
		lifetime: lifetime,
	}
	return nil
}

func (m *mockRouter) addedMapping() mapping {
	m.Lock()
	defer m.Unlock()

	return m.mapping
}

func (m *mockRouter) DeleteMapping(protocol string, extport, intport int) error {
	return nil
}

func (m *mockRouter) ExternalIP() (net.IP, error) {
	return nil, nil
}

func (m *mockRouter) String() string {
	return ""
}
