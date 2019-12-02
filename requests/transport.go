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
)

// GetDefaultTransport returns default HTTP transport which
// should be reused as it caches underlying TCP connections.
// If connections pooling is not needed consider to set
// DisableKeepAlives=false and MaxIdleConnsPerHost=-1.
func GetDefaultTransport(srcIP string) *http.Transport {
	ipAddress := net.ParseIP(srcIP)
	localIPAddress := &net.TCPAddr{IP: ipAddress}

	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: localIPAddress,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
