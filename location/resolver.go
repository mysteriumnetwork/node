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
	"github.com/oschwald/geoip2-golang"
	"net"
)

type resolver struct {
	databasePath string
}

// NewResolver returns Resolver which uses country database
func NewResolver(databasePath string) Resolver {
	return &resolver{
		databasePath: databasePath,
	}
}

// ResolveCountry maps given ip to country
func (r *resolver) ResolveCountry(ip string) (string, error) {
	db, err := geoip2.Open(r.databasePath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	ipObject := net.ParseIP(ip)
	if ipObject == nil {
		return "", errors.New("failed to parse IP")
	}

	countryRecord, err := db.Country(ipObject)
	if err != nil {
		return "", err
	}

	country := countryRecord.Country.IsoCode
	if country == "" {
		country = countryRecord.RegisteredCountry.IsoCode
		if country == "" {
			return "", errors.New("failed to resolve country")
		}
	}

	return country, nil
}
