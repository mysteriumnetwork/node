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
	"errors"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/stretchr/testify/assert"
)

func TestLocationCacheFirstCall(t *testing.T) {
	locationDetector := NewStaticResolver("country", "city", "residential", ip.NewResolverMock("100.100.100.100"))
	locationCache := NewLocationCache(locationDetector)

	location := locationCache.Get()
	assert.Equal(t, Location{}, location)
}

func TestLocationCacheFirstSecondCalls(t *testing.T) {
	locationDetector := NewStaticResolver("country", "city", "residential", ip.NewResolverMock("100.100.100.100"))
	locationCache := NewLocationCache(locationDetector)

	location, err := locationCache.RefreshAndGet()
	assert.Equal(t, "country", location.Country)
	assert.Equal(t, "100.100.100.100", location.IP)
	assert.NoError(t, err)

	locationSecondCall := locationCache.Get()
	assert.Equal(t, location, locationSecondCall)
}

func TestLocationCacheWithError(t *testing.T) {
	locationErr := errors.New("location DBResolver error")
	locationDetector := NewFailingResolver(locationErr)
	locationCache := NewLocationCache(locationDetector)

	location, err := locationCache.RefreshAndGet()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, Location{}, location)
}

func TestProperCache_needsRefresh(t *testing.T) {
	type fields struct {
		lastFetched time.Time
		expiry      time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "returns true on zero time",
			want:   true,
			fields: fields{},
		},
		{
			name: "returns true if expired",
			want: true,
			fields: fields{
				lastFetched: time.Now().Add(time.Second * -59),
				expiry:      time.Minute * 1,
			},
		},
		{
			name: "returns false if updated recently",
			want: false,
			fields: fields{
				lastFetched: time.Now().Add(time.Second * -61),
				expiry:      time.Minute * 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ProperCache{
				lastFetched: tt.fields.lastFetched,
				expiry:      tt.fields.expiry,
			}
			if got := c.needsRefresh(); got != tt.want {
				t.Errorf("ProperCache.needsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}
