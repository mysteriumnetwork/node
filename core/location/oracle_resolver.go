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
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/requests"
)

type oracleResolver struct {
	httpClient *requests.HTTPClient
	address    string
}

type oracleLocation struct {
	ASN       int    `json:"asn"`
	City      string `json:"city"`
	Region    string `json:"region"`
	Continent string `json:"continent"`
	Country   string `json:"country"`
	IP        string `json:"ip"`
	ISP       string `json:"isp"`
	NodeType  string `json:"node_type"`
}

func (l oracleLocation) ToLocation() locationstate.Location {
	return locationstate.Location{
		ASN:       l.ASN,
		City:      l.City,
		Region:    l.Region,
		Continent: l.Continent,
		Country:   l.Country,
		IP:        l.IP,
		ISP:       l.ISP,
		IPType:    l.NodeType,
	}
}

// NewOracleResolver returns new db resolver initialized from Location Oracle service
func NewOracleResolver(httpClient *requests.HTTPClient, address string) *oracleResolver {
	return &oracleResolver{
		httpClient: httpClient,
		address:    address,
	}
}

// DetectLocation detects current IP-address provides location information for the IP.
func (o *oracleResolver) DetectLocation() (location locationstate.Location, err error) {
	log.Debug().Msg("Detecting with oracle resolver")
	request, err := requests.NewGetRequest(o.address, "", nil)
	if err != nil {
		log.Error().Err(err).Msg("")
		return locationstate.Location{}, errors.Wrap(err, "failed to create request")
	}

	var res oracleLocation
	err = o.httpClient.DoRequestAndParseResponse(request, &res)

	return res.ToLocation(), errors.Wrap(err, "failed to execute request")
}

// DetectProxyLocation detects current IP-address provides location information for the IP.
func (o *oracleResolver) DetectProxyLocation(proxyPort int) (location locationstate.Location, err error) {
	log.Debug().Msg("Detecting with oracle resolver")
	request, err := requests.NewGetRequest(o.address, "", nil)
	if err != nil {
		log.Error().Err(err).Msg("")
		return locationstate.Location{}, errors.Wrap(err, "failed to create request")
	}

	var res oracleLocation
	err = o.httpClient.DoRequestViaProxyAndParseResponse(request, &res, proxyPort)

	return res.ToLocation(), errors.Wrap(err, "failed to execute request")
}
