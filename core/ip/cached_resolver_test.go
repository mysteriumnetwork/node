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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCachedResolverCachesPublicIP(t *testing.T) {
	tests := []struct {
		name              string
		cacheDuration     time.Duration
		sleepAfterCall    time.Duration
		expectedIP        string
		expectedIPFetches int
	}{
		{
			name:              "Test ip is fetch and cache",
			cacheDuration:     50 * time.Millisecond,
			sleepAfterCall:    time.Microsecond,
			expectedIP:        "1.1.1.1",
			expectedIPFetches: 1,
		},
		{
			name:              "Test ip cache update when cache duration has passed",
			cacheDuration:     time.Microsecond,
			sleepAfterCall:    time.Millisecond,
			expectedIP:        "1.1.1.1",
			expectedIPFetches: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mr := &mockRealResolver{}
			cr := NewCachedResolver(mr, test.cacheDuration)

			var actualIP string
			var err error
			for i := 0; i < 5; i++ {
				actualIP, err = cr.GetPublicIP()
				assert.NoError(t, err)
				time.Sleep(test.sleepAfterCall)
			}

			assert.Equal(t, test.expectedIP, actualIP)
			assert.Equal(t, test.expectedIPFetches, mr.getPublicIPCalls)
		})
	}
}

func TestCachedResolverCachesOutboundIP(t *testing.T) {
	tests := []struct {
		name              string
		cacheDuration     time.Duration
		ipCheckInterval   time.Duration
		expectedIP        string
		expectedIPFetches int
	}{
		{
			name: "Test ip is fetch and cache",
			// monotonic time resolution is fairly low on some OS'es, such as 15ms on Windows 2008
			cacheDuration:     50 * time.Millisecond,
			ipCheckInterval:   time.Microsecond,
			expectedIP:        "192.168.1.2",
			expectedIPFetches: 1,
		},
		{
			name:              "Test ip cache update when cache duration has passed",
			cacheDuration:     time.Microsecond,
			ipCheckInterval:   time.Millisecond,
			expectedIP:        "192.168.1.2",
			expectedIPFetches: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mr := &mockRealResolver{}
			cr := NewCachedResolver(mr, test.cacheDuration)

			var actualIP string
			var err error
			for i := 0; i < 5; i++ {
				actualIP, err = cr.GetOutboundIP()
				assert.NoError(t, err)
				time.Sleep(test.ipCheckInterval)
			}

			assert.Equal(t, test.expectedIP, actualIP)
			assert.Equal(t, test.expectedIPFetches, mr.getOutboundIPCalls)
		})
	}
}

type mockRealResolver struct {
	getOutboundIPCalls int
	getPublicIPCalls   int
}

func (m *mockRealResolver) GetOutboundIP() (string, error) {
	m.getOutboundIPCalls++
	return "192.168.1.2", nil
}

func (m *mockRealResolver) GetPublicIP() (string, error) {
	m.getPublicIPCalls++
	return "1.1.1.1", nil
}

func (m *mockRealResolver) GetProxyIP(_ int) (string, error) {
	return m.GetPublicIP()
}
