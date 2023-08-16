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

package tequil

import "strings"

// TequilapiURLPrefix tequilapi reverse proxy prefix
const TequilapiURLPrefix = "/tequilapi"

// UnprotectedRoutes these routes are not protected by reverse proxy
var UnprotectedRoutes = []string{"/auth/authenticate", "/auth/login", "/healthcheck", "/config/user", "/config/ui/features"}

// IsUnprotectedRoute helper method for checking if route is unprotected
func IsUnprotectedRoute(url string) bool {
	for _, route := range UnprotectedRoutes {
		if strings.Contains(url, route) {
			return true
		}
	}

	return false
}

// IsProtectedRoute helper method for checking if route is protected
func IsProtectedRoute(url string) bool {
	return !IsUnprotectedRoute(url)
}

// IsReverseProxyRoute helper method for checking if URL is of tequilapi reverse proxy
func IsReverseProxyRoute(url string) bool {
	return strings.Contains(url, TequilapiURLPrefix)
}
