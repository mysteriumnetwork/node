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

package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenFromContext retrieve token from request Header or Cookie
func TokenFromContext(c *gin.Context) (string, error) {
	token, err := fromHeader(c)
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}

	return fromCookie(c)
}

func fromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", nil // No error, just no token
	}

	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New(`authorization header format must be: "Bearer {token}"`)
	}

	return authHeaderParts[1], nil
}

func fromCookie(c *gin.Context) (string, error) {
	token, err := c.Cookie(JWTCookieName)
	if err == http.ErrNoCookie {
		// No error, just no token
		return "", nil
	}
	return token, nil
}
