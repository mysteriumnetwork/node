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

package ui

import (
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/tequilapi/endpoints"
)

func buildTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 20 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     15,
	}
}

func buildReverseProxy(transport *http.Transport, tequilapiPort int) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = tequilapiHost + ":" + strconv.Itoa(tequilapiPort)
			req.URL.Path = strings.Replace(req.URL.Path, tequilapiUrlPrefix, "", 1)
			req.URL.Path = strings.TrimRight(req.URL.Path, "/")
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Set("Access-Control-Allow-Origin", "*")
			res.Header.Set("Access-Control-Allow-Headers", "*")
			res.Header.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			return nil
		},
		Transport: transport,
	}

	proxy.FlushInterval = 10 * time.Millisecond

	return proxy
}

// ReverseTequilapiProxy proxies UIServer requests to the TequilAPI server
func ReverseTequilapiProxy(tequilapiPort int, authenticator jwtAuthenticator) gin.HandlerFunc {
	proxy := buildReverseProxy(buildTransport(), tequilapiPort)

	return func(c *gin.Context) {
		// skip non Tequilapi routes
		if !isTequilapiURL(c.Request.URL.Path) {
			return
		}

		// authenticate all but the login route
		if !isTequilapiURL(c.Request.URL.Path, endpoints.TequilapiLoginEndpointPath) {
			cookieToken, err := c.Cookie(auth.JWTCookieName)

			if err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			if _, err := authenticator.ValidateToken(cookieToken); err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}

		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					// ignore streaming errors (SSE)
					// there's nothing we can do about them
				} else {
					panic(err)
				}
			}
		}()

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func isTequilapiURL(url string, endpoints ...string) bool {
	return strings.Contains(url, tequilapiUrlPrefix+strings.Join(endpoints, ""))
}
