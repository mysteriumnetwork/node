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

package location

import (
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
)

type oracleResolver struct {
	http                  requests.HTTPTransport
	oracleResolverAddress string
}

// NewOracleResolver returns new db resolver initialized from Location Oracle service
func NewOracleResolver(srcIP, address string) *oracleResolver {
	return &oracleResolver{
		requests.NewHTTPClient(srcIP, 20*time.Second),
		address,
	}
}

// DetectLocation detects current IP-address provides location information for the IP.
func (o *oracleResolver) DetectLocation() (location Location, err error) {
	log.Debug("detecting with oracle resolver")
	request, err := requests.NewGetRequest(o.oracleResolverAddress, "", nil)
	if err != nil {
		log.Error(err)
		return Location{}, errors.Wrap(err, "failed to create request")
	}

	err = o.http.DoRequestAndParseResponse(request, &location)
	return location, errors.Wrap(err, "failed to execute request")
}
