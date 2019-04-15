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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

const oracleResolverLogPrefix = "[location.Oracle.Resolver] "

type oracleResolver struct {
	http                  mysterium.HTTPTransport
	oracleResolverAddress string
}

// NewOracleResolver returns new db resolver initialized from Location Oracle service
func NewOracleResolver(address string) *oracleResolver {
	return &oracleResolver{
		newHTTPTransport(1 * time.Minute),
		address,
	}
}

// DetectLocation detects current IP-address provides location information for the IP.
func (o *oracleResolver) DetectLocation() (location Location, err error) {
	return o.ResolveLocation(nil)
}

// ResolveLocation maps given ip to country.
func (o *oracleResolver) ResolveLocation(ip net.IP) (location Location, err error) {
	var ipAddress string
	if ip != nil {
		ipAddress = ip.String()
	}

	request, err := http.NewRequest("GET", o.oracleResolverAddress+"/"+ipAddress, nil)
	if err != nil {
		log.Error(oracleResolverLogPrefix, err)
		return Location{}, err
	}

	err = o.doRequest(request, &location)
	return location, err
}

func (o *oracleResolver) doRequest(request *http.Request, responseDto interface{}) error {
	response, err := o.http.Do(request)
	if err != nil {
		log.Error(oracleResolverLogPrefix, err)
		return err
	}
	defer response.Body.Close()

	err = parseResponseError(response)
	if err != nil {
		log.Error(oracleResolverLogPrefix, err)
		return err
	}

	return parseResponseJSON(response, &responseDto)
}

func newHTTPTransport(requestTimeout time.Duration) mysterium.HTTPTransport {
	return &http.Client{
		Transport: &http.Transport{
			//Don't reuse tcp connections for request - see ip/rest_resolver.go for details
			DisableKeepAlives: true,
		},
		Timeout: requestTimeout,
	}
}

func parseResponseJSON(response *http.Response, dto interface{}) error {
	responseJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(responseJSON, dto)
	if err != nil {
		return err
	}

	return nil
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("server response invalid: %s (%s)", response.Status, response.Request.URL)
	}

	return nil
}
