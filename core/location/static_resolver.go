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
	"net"

	"github.com/mysteriumnetwork/node/core/ip"
)

// StaticResolver struct represents country by ip ExternalDBResolver which always returns specified country
type StaticResolver struct {
	country  string
	city     string
	nodeType string
	err      error

	ipResolver ip.Resolver
}

// NewStaticResolver creates new StaticResolver with specified country
func NewStaticResolver(country, city, nodeType string, ipResolver ip.Resolver) *StaticResolver {
	return &StaticResolver{
		country:  country,
		city:     city,
		nodeType: nodeType,
		err:      nil,

		ipResolver: ipResolver,
	}
}

// NewFailingResolver returns StaticResolver with entered error
func NewFailingResolver(err error) *StaticResolver {
	return &StaticResolver{
		err:        err,
		ipResolver: ip.NewResolverMock(""),
	}
}

// DetectLocation maps given ip to country
func (d *StaticResolver) DetectLocation(ip net.IP) (Location, error) {
	pubIP, err := d.ipResolver.GetPublicIP()
	if err != nil {
		return Location{}, err
	}

	return Location{
		Country:  d.country,
		City:     d.city,
		NodeType: d.nodeType,
		IP:       pubIP,
	}, d.err
}
