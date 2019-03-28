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

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/stretchr/testify/assert"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetectorFake("8.8.8.8", "US")

	location, err := detector.DetectLocation()
	assert.Equal(t, "US", location.Country)
	assert.Equal(t, "8.8.8.8", location.IP)
	assert.NoError(t, err)
}

func TestWithIpResolverFailing(t *testing.T) {
	ipErr := errors.New("ip DbResolver error")
	detector := NewDetector(
		ip.NewResolverMockFailing(ipErr),
		NewStaticResolver(""),
	)

	location, err := detector.DetectLocation()
	assert.EqualError(t, ipErr, err.Error())
	assert.Equal(t, Location{}, location)
}

func TestWithLocationResolverFailing(t *testing.T) {
	locationErr := errors.New("location DbResolver error")
	detector := NewDetector(
		ip.NewResolverMock(""),
		NewFailingResolver(locationErr),
	)

	location, err := detector.DetectLocation()
	assert.EqualError(t, locationErr, err.Error())
	assert.Equal(t, Location{}, location)
}
