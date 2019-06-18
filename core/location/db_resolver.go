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
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
)

// DBResolver struct represents ip -> country resolver which uses geoip2 data reader
type DBResolver struct {
	dbReader   *geoip2.Reader
	ipResolver ip.Resolver
}

// NewExternalDBResolver returns Resolver which uses external country database
func NewExternalDBResolver(databasePath string, ipResolver ip.Resolver) (*DBResolver, error) {
	db, err := geoip2.Open(databasePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open external db")
	}

	return &DBResolver{
		dbReader:   db,
		ipResolver: ipResolver,
	}, nil
}

// DetectLocation detects current IP-address provides location information for the IP.
func (r *DBResolver) DetectLocation() (loc Location, err error) {
	log.Debug("detecting with DB resolver")
	ipAddress, err := r.ipResolver.GetPublicIP()
	if err != nil {
		return Location{}, errors.Wrap(err, "failed to get public IP")
	}

	ip := net.ParseIP(ipAddress)
	countryRecord, err := r.dbReader.Country(ip)
	if err != nil {
		return loc, errors.Wrap(err, "failed to get a country")
	}

	country := countryRecord.Country.IsoCode
	if country == "" {
		country = countryRecord.RegisteredCountry.IsoCode
		if country == "" {
			return loc, errors.New("failed to resolve country")
		}
	}

	loc.IP = ip.String()
	loc.Country = country
	return loc, nil
}
