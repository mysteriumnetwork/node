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

	"github.com/mysteriumnetwork/node/tequilapi/tequil"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/core/auth"
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

func buildReverseProxy(tequilapiAddress string, tequilapiPort int) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = tequilapiAddress + ":" + strconv.Itoa(tequilapiPort)
			req.URL.Path = strings.Replace(req.URL.Path, tequil.TequilapiURLPrefix, "", 1)
			req.URL.Path = strings.TrimRight(req.URL.Path, "/")
			req.Header.Del("Origin")
			req.Host = "127.0.0.1" + ":" + strconv.Itoa(tequilapiPort)
		},
		ModifyResponse: func(res *http.Response) error {
			// remove TequilAPI CORS headers
			// these will be overwritten by Gin middleware
			res.Header.Del("Access-Control-Allow-Origin")
			res.Header.Del("Access-Control-Allow-Headers")
			res.Header.Del("Access-Control-Allow-Methods")
			return nil
		},
		Transport: buildTransport(),
	}

	proxy.FlushInterval = 10 * time.Millisecond

	return proxy
}

// ReverseTequilapiProxy proxies UIServer requests to the TequilAPI server
func ReverseTequilapiProxy(tequilapiAddress string, tequilapiPort int, authenticator jwtAuthenticator) gin.HandlerFunc {
	proxy := buildReverseProxy(tequilapiAddress, tequilapiPort)

	return func(c *gin.Context) {
		// skip non reverse proxy routes
		if !tequil.IsReverseProxyRoute(c.Request.URL.Path) {
			return
		}

		// authenticate all but the authentication routes
		if tequil.IsProtectedRoute(c.Request.URL.Path) {
			authToken, err := auth.TokenFromContext(c)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			if _, err := authenticator.ValidateToken(authToken); err != nil {
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
