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

	"github.com/mysterium/node/ip"
	"github.com/stretchr/testify/assert"
)

func TestNewDetector(t *testing.T) {
	ipResolver := ip.NewFakeResolver("8.8.8.8")
	detector := NewDetector(ipResolver, "../bin/common_package/GeoLite2-Country.mmdb")
	location, err := detector.DetectLocation()
	assert.Equal(t, "US", location.Country)
	assert.Equal(t, "8.8.8.8", location.IP)
	assert.NoError(t, err)
}

func TestWithIpResolverFailing(t *testing.T) {
	ipErr := errors.New("ip resolver error")
	ipResolver := ip.NewFailingFakeResolver(ipErr)
	detector := NewDetectorWithLocationResolver(ipResolver, NewResolverFake(""))
	location, err := detector.DetectLocation()
	assert.EqualError(t, ipErr, err.Error())
	assert.Equal(t, Location{}, location)
}

func TestWithLocationResolverFailing(t *testing.T) {
	ipResolver := ip.NewFakeResolver("")
	locationErr := errors.New("location resolver error")
	locationResolver := NewFailingResolverFake(locationErr)
	detector := NewDetectorWithLocationResolver(ipResolver, locationResolver)
	location, err := detector.DetectLocation()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, Location{}, location)
}
