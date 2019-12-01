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

package ip

import (
	"net"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/requests"
)

const apiClient = "goclient-v0.1"

// Resolver allows resolving current public and outbound IPs
type Resolver interface {
	GetOutboundIPAsString() (string, error)
	GetOutboundIP() (net.IP, error)
	GetPublicIP() (string, error)
}

// ResolverImpl represents data required to operate resolving
type ResolverImpl struct {
	bindAddress string
	url         string
	http        *requests.HTTPClient
}

// NewResolver creates new ip-detector resolver with default timeout of one minute
func NewResolver(httpClient *requests.HTTPClient, bindAddress, url string) *ResolverImpl {
	return &ResolverImpl{
		bindAddress: bindAddress,
		url:         url,
		http:        httpClient,
	}
}

type ipResponse struct {
	IP string `json:"IP"`
}

// declared as var for override in test
var checkAddress = "8.8.8.8:53"

// GetOutboundIPAsString returns current outbound IP as string for current system
func (r *ResolverImpl) GetOutboundIPAsString() (string, error) {
	ip, err := r.GetOutboundIP()
	if err != nil {
		return "", nil
	}
	return ip.String(), nil
}

// GetOutboundIP returns current outbound IP for current system
func (r *ResolverImpl) GetOutboundIP() (net.IP, error) {
	ipAddress := net.ParseIP(r.bindAddress)
	localIPAddress := net.UDPAddr{IP: ipAddress}

	dialer := net.Dialer{LocalAddr: &localIPAddress}

	conn, err := dialer.Dial("udp4", checkAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine outbound IP")
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP, nil
}

// GetPublicIP returns current public IP
func (r *ResolverImpl) GetPublicIP() (string, error) {
	var ipResponse ipResponse

	request, err := requests.NewGetRequest(r.url, "", nil)
	request.Header.Set("User-Agent", apiClient)
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Error().Err(err).Msg("")
		return "", err
	}

	err = r.http.DoRequestAndParseResponse(request, &ipResponse)
	if err != nil {
		return "", err
	}

	log.Debug().Msg("IP detected: " + ipResponse.IP)
	return ipResponse.IP, nil
}
