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

package middlewares

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/utils/domain"
)

// ApplyCacheConfigMiddleware forces no caching policy via "Cache-control" header
func ApplyCacheConfigMiddleware(ctx *gin.Context) {
	ctx.Writer.Header().Set("Cache-control", strings.Join([]string{"no-cache", "no-store", "must-revalidate"}, ", "))
}

// NewHostFilter returns instance of middleware allowing only requests
// with allowed domains in Host header
func NewHostFilter() func(*gin.Context) {
	whitelist := domain.NewWhitelist(
		strings.Split(config.GetString(config.FlagTequilapiAllowedHostnames), ","))
	return func(c *gin.Context) {
		host := c.Request.Host
		if host == "" {
			return
		}

		hostname, _, err := net.SplitHostPort(host)
		if err != nil {
			// There was no port, so we assume the address was just a hostname
			hostname = host
		}

		if net.ParseIP(hostname) != nil {
			return
		}

		if whitelist.Match(hostname) {
			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}

// NewLocalhostOnlyFilter returns instance of middleware allowing only requests
// with local client IP.
func NewLocalhostOnlyFilter() func(*gin.Context) {
	return func(c *gin.Context) {

		// ClientIP() parses the headers defined in Engine.RemoteIPHeaders if there is
		// so it handles clients behind proxy
		isLocal := net.ParseIP(c.ClientIP()).IsLoopback()
		if isLocal {
			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
