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

package quality

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
)

// NewElasticSearchTransport creates transport allowing to send events to ElasticSearch through HTTP
func NewElasticSearchTransport(srcIP, url string, timeout time.Duration) Transport {
	return &elasticSearchTransport{
		http: requests.NewHTTPClient(srcIP, timeout),
		url:  url,
	}
}

type elasticSearchTransport struct {
	http requests.HTTPTransport
	url  string
}

func (transport *elasticSearchTransport) SendEvent(event Event) error {
	req, err := requests.NewPostRequest(transport.url, "/", event)
	if err != nil {
		return err
	}

	response, err := transport.http.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrapf(err, "error while reading response body")
	}
	body := string(bodyBytes)

	if response.StatusCode != 200 {
		return errors.Errorf("unexpected response status: %v, body: %v", response.Status, body)
	}

	if strings.ToUpper(body) != "OK" {
		return errors.Errorf("unexpected response body: %v", body)
	}

	return nil
}
