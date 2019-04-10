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
	"net"

	"github.com/oschwald/geoip2-golang"
)

// DbResolver struct represents ip -> country resolver which uses geoip2 data reader
type DbResolver struct {
	dbReader *geoip2.Reader
}

// NewExternalDBResolver returns Resolver which uses external country database
func NewExternalDBResolver(databasePath string) Resolver {
	db, err := geoip2.Open(databasePath)
	if err != nil {
		return NewFailingResolver(err)
	}

	return &DbResolver{
		dbReader: db,
	}
}

// ResolveCountry maps given ip to country
func (r *DbResolver) ResolveCountry(ip net.IP) (string, error) {
	location, err := r.DetectLocation(ip)
	if err != nil {
		return "", err
	}

	return location.Country, nil
}

func (r *DbResolver) DetectLocation(ip net.IP) (Location, error) {
	if ip == nil {
		return Location{}, errors.New("failed to parse IP")
	}

	countryRecord, err := r.dbReader.Country(ip)
	if err != nil {
		return Location{}, err
	}

	country := countryRecord.Country.IsoCode
	if country == "" {
		country = countryRecord.RegisteredCountry.IsoCode
		if country == "" {
			return Location{}, errors.New("failed to resolve country")
		}
	}

	return Location{Country: country}, nil
}
