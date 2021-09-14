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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CorsPolicy resolves allowed origin
type CorsPolicy interface {
	AllowedOrigin(requestOrigin string) string
}

// ApplyCORSMiddleware handles Access-Control-Allow-Origin header for tequilAPI
func ApplyCORSMiddleware(policy CorsPolicy) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		if isPreflightCorsRequest(ctx) {
			generatePreflightResponse(ctx, policy)
			return
		}

		allowCorsActions(ctx, policy)
	}
}

func allowCorsActions(ctx *gin.Context, policy CorsPolicy) {
	requestOrigin := ctx.Request.Header.Get("Origin")
	allowedOrigin := policy.AllowedOrigin(requestOrigin)

	ctx.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	ctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

func isPreflightCorsRequest(ctx *gin.Context) bool {
	isOptionsMethod := ctx.Request.Method == http.MethodOptions
	containsAccessControlRequestMethod := ctx.Request.Header.Get("Access-Control-Request-Method") != ""
	containsOriginHeader := ctx.Request.Header.Get("Origin") != ""
	return isOptionsMethod && containsOriginHeader && containsAccessControlRequestMethod
}

func generatePreflightResponse(ctx *gin.Context, policy CorsPolicy) {
	allowCorsActions(ctx, policy)
	//allow all headers which were defined in preflight request
	for _, headerValue := range ctx.Request.Header["Access-Control-Request-Headers"] {
		ctx.Writer.Header().Add("Access-Control-Allow-Headers", headerValue)
	}
	ctx.Writer.WriteHeader(http.StatusNoContent)
}

// ApplyCacheConfigMiddleware forces no caching policy via "Cache-control" header
func ApplyCacheConfigMiddleware(ctx *gin.Context) {
	ctx.Writer.Header().Set("Cache-control", strings.Join([]string{"no-cache", "no-store", "must-revalidate"}, ", "))
}
