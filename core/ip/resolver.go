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
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/requests"
)

const apiClient = "goclient-v0.1"

// Resolver allows resolving current public and outbound IPs
type Resolver interface {
	GetOutboundIP() (string, error)
	GetPublicIP() (string, error)
	GetProxyIP(proxyPort int) (string, error)
}

// ResolverImpl represents data required to operate resolving
type ResolverImpl struct {
	bindAddress string
	url         string
	httpClient  *requests.HTTPClient
	fallbacks   []string
}

// NewResolver creates new ip-detector resolver with default timeout of one minute
func NewResolver(httpClient *requests.HTTPClient, bindAddress, url string, fallbacks []string) *ResolverImpl {
	return &ResolverImpl{
		bindAddress: bindAddress,
		url:         url,
		httpClient:  httpClient,
		fallbacks:   fallbacks,
	}
}

type ipResponse struct {
	IP string `json:"IP"`
}

// declared as var for override in test
var checkAddress = "8.8.8.8:53"

// GetOutboundIP returns current outbound IP as string for current system
func (r *ResolverImpl) GetOutboundIP() (string, error) {
	ip, err := r.getOutboundIP()
	if err != nil {
		return "", nil
	}
	return ip.String(), nil
}

func (r *ResolverImpl) getOutboundIP() (net.IP, error) {
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
	if err != nil {
		log.Error().Err(err).Msg("")
		return "", err
	}
	request.Header.Set("User-Agent", apiClient)
	request.Header.Set("Accept", "application/json")

	err = r.httpClient.DoRequestAndParseResponse(request, &ipResponse)
	if err != nil {
		log.Err(err).Msg("could not reach location service, will use fallbacks")
		return r.findPublicIPViaFallbacks()
	}

	return ipResponse.IP, nil
}

// GetProxyIP returns proxy public IP
func (r *ResolverImpl) GetProxyIP(proxyPort int) (string, error) {
	var ipResponse ipResponse

	request, err := requests.NewGetRequest(r.url, "", nil)
	if err != nil {
		log.Error().Err(err).Msg("")
		return "", err
	}

	request.Header.Set("User-Agent", apiClient)
	request.Header.Set("Accept", "application/json")

	err = r.httpClient.DoRequestViaProxyAndParseResponse(request, &ipResponse, proxyPort)
	if err != nil {
		log.Err(err).Msg("could not reach location service, will use fallbacks")
		return r.findPublicIPViaFallbacks()
	}

	return ipResponse.IP, nil
}

func (r *ResolverImpl) findPublicIPViaFallbacks() (string, error) {
	// To prevent blocking for a long time on a service that might be dead, use the following fallback mechanic:
	// Choose 3 fallback addresses at random and execute lookups on them in parallel.
	// Return the first successful result or an error if such occurs.
	// This prevents providers from not being able to provide sessions due to not having a fresh public IP address.
	desiredLength := 3
	res := make(chan string, desiredLength)
	wg := sync.WaitGroup{}
	wg.Add(desiredLength)

	go func() {
		wg.Wait()
		close(res)
	}()

	for _, v := range shuffleStringSlice(r.fallbacks)[:desiredLength] {
		go func(url string) {
			defer wg.Done()
			r, err := RequestAndParsePlainIPResponse(r.httpClient, url)
			if err != nil {
				log.Err(err).Str("url", url).Msg("public ip fallback error")
			}
			res <- r
		}(v)
	}

	for ip := range res {
		if ip != "" {
			return ip, nil
		}
	}

	return "", errors.New("out of fallbacks")
}
