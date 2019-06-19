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

package tequilapi

import (
	"net/http"
	"strings"
)

type corsHandler struct {
	originalHandler http.Handler
	corsPolicy      CorsPolicy
}

// CorsPolicy resolves allowed origin
type CorsPolicy interface {
	AllowedOrigin(requestOrigin string) string
}

func (wrapper corsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if isPreflightCorsRequest(req) {
		generatePreflightResponse(req, resp, wrapper.corsPolicy)
		return
	}

	allowCorsActions(resp, req, wrapper.corsPolicy)
	wrapper.originalHandler.ServeHTTP(resp, req)
}

// ApplyCors wraps original handler by adding cors headers to response BEFORE original ServeHTTP method is called
func ApplyCors(original http.Handler, corsPolicy CorsPolicy) http.Handler {
	return corsHandler{originalHandler: original, corsPolicy: corsPolicy}
}

func allowCorsActions(resp http.ResponseWriter, _ *http.Request, _ CorsPolicy) {
	//requestOrigin := req.Header.Get("Origin")
	//allowedOrigin := corsPolicy.AllowedOrigin(requestOrigin)

	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

func isPreflightCorsRequest(req *http.Request) bool {
	isOptionsMethod := req.Method == http.MethodOptions
	containsAccessControlRequestMethod := req.Header.Get("Access-Control-Request-Method") != ""
	containsOriginHeader := req.Header.Get("Origin") != ""
	return isOptionsMethod && containsOriginHeader && containsAccessControlRequestMethod
}

func generatePreflightResponse(req *http.Request, resp http.ResponseWriter, corsPolicy CorsPolicy) {
	allowCorsActions(resp, req, corsPolicy)
	//allow all headers which were defined in preflight request
	for _, headerValue := range req.Header["Access-Control-Request-Headers"] {
		resp.Header().Add("Access-Control-Allow-Headers", headerValue)
	}
}

const cacheControlHeader = "Cache-control"
const noStore = "no-store"
const noCache = "no-cache"
const mustRevalidate = "must-revalidate"

type cacheControl struct {
	originalHandler http.Handler
}

func (cc cacheControl) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set(cacheControlHeader, strings.Join([]string{noCache, noStore, mustRevalidate}, ", "))
	cc.originalHandler.ServeHTTP(resp, req)
}

// DisableCaching middleware adds cache disabling headers to http response
func DisableCaching(original http.Handler) http.Handler {
	return &cacheControl{
		original,
	}
}
