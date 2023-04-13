/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/node/core/auth"
)

var unprotectedRoutes = []string{"/auth/authenticate", "/auth/login", "/healthcheck"}

type jwtAuthenticator interface {
	ValidateToken(token string) (bool, error)
}

// ApplyMiddlewareTokenAuth creates token authenticator
func ApplyMiddlewareTokenAuth(authenticator jwtAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isUnprotected(c.Request.URL.Path) {
			return
		}

		token, err := auth.TokenFromContext(c)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if _, err := authenticator.ValidateToken(token); err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}

func isUnprotected(url string) bool {
	for _, route := range unprotectedRoutes {
		if strings.Contains(url, route) {
			return true
		}
	}

	return false
}
