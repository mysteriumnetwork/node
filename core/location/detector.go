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
	"github.com/mysteriumnetwork/node/core/ip"
)

// NewDetector constructs Detector instance
func NewDetector(ipResolver ip.Resolver, locationResolver Resolver) Detector {
	return &detector{
		ipResolver:       ipResolver,
		locationResolver: locationResolver,
	}
}

// NewDetectorFake constructs Detector instance with faked data
func NewDetectorFake(ipAddress string, country string) Detector {
	return &detector{
		ipResolver:       ip.NewResolverFake(ipAddress),
		locationResolver: NewStaticResolver(country),
	}
}

// NewDetectorFakeFailing constructs Detector instance with faked error
func NewDetectorFakeFailing(err error) Detector {
	return &detector{
		ipResolver:       ip.NewResolverFake(""),
		locationResolver: NewFailingResolver(err),
	}
}

type detector struct {
	ipResolver       ip.Resolver
	locationResolver Resolver
}

// Maps current ip to country
func (d *detector) DetectLocation() (Location, error) {
	ipAddress, err := d.ipResolver.GetPublicIP()
	if err != nil {
		return Location{}, err
	}

	country, err := d.locationResolver.ResolveCountry(ipAddress)
	if err != nil {
		return Location{}, err
	}

	location := Location{Country: country, IP: ipAddress}
	return location, nil
}
