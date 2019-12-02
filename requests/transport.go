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

package requests

import (
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	maxHTTPRetries  = 3
	delayAfterRetry = 50 * time.Millisecond
)

func newTransport(localIPAddress *net.TCPAddr) *transport {
	return &transport{
		rt: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 30 * time.Second,
				LocalAddr: localIPAddress,
			}).DialContext,
			ForceAttemptHTTP2:   true,
			MaxIdleConns:        300,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 30 * time.Second,
		},
	}
}

type transport struct {
	rt http.RoundTripper
}

// RoundTrip does HTTP requests with retries. Retries are needed since
// error can happen when HTTP client closes idle connections.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for i := 1; i <= maxHTTPRetries; i++ {
		res, err := t.rt.RoundTrip(req)
		if err == nil {
			return res, nil
		}

		log.Warn().Err(err).Msgf("Failed to call %q. Retrying.", req.URL.String())
		time.Sleep(delayAfterRetry)
		if i == maxHTTPRetries {
			return res, err
		}
	}
	return nil, nil
}
