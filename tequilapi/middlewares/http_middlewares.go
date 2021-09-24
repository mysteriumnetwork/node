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
)

// ApplyCacheConfigMiddleware forces no caching policy via "Cache-control" header
func ApplyCacheConfigMiddleware(ctx *gin.Context) {
	ctx.Writer.Header().Set("Cache-control", strings.Join([]string{"no-cache", "no-store", "must-revalidate"}, ", "))
}

func normalizeHostname(hostname string) string {
	return strings.ToLower(
		strings.TrimRight(
			strings.TrimSpace(hostname),
			".",
		),
	)
}

// NewHostFilter returns instance of middleware allowing only requests
// with allowed domains in Host header
func NewHostFilter() func(*gin.Context) {
	domainList := strings.Split(config.GetString(config.FlagTequilapiAllowedHostnames), ",")
	exactList := make(map[string]struct{})
	suffixList := make(map[string]struct{})
	for _, domain := range domainList {
		domain = strings.TrimSpace(domain)
		normalized := normalizeHostname(domain)
		if strings.Index(domain, ".") == 0 {
			// suffix pattern
			suffixList[strings.TrimLeft(normalized, ".")] = struct{}{}
		} else {
			// exact domain name
			exactList[normalized] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		// handle special case of root suffix (".")
		if _, found := suffixList[""] ; found {
			return
		}

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

		hostname = normalizeHostname(hostname)

		// check for exact match
		if _, found := exactList[hostname]; found {
			return
		}

		// check for suffix match
		for needle := strings.Split(hostname, ".")[1:] ; len(needle) > 0 ; needle = needle[1:] {
			if _, found := suffixList[strings.Join(needle, ".")] ; found {
				return
			}
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
