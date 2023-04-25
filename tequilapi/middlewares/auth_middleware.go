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

	"github.com/mysteriumnetwork/node/tequilapi/tequil"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/node/core/auth"
)

type jwtAuthenticator interface {
	ValidateToken(token string) (bool, error)
}

// ApplyMiddlewareTokenAuth creates token authenticator
func ApplyMiddlewareTokenAuth(authenticator jwtAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tequil.IsUnprotectedRoute(c.Request.URL.Path) {
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
