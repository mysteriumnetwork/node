package tequilapi

import (
	"net/http"
)

type corsHandler struct {
	originalHandler http.Handler
}

func (wrapper corsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if isPreflightCorsRequest(req) {
		generatePreflightResponse(req, resp)
		return
	}

	allowAllCorsActions(resp)
	wrapper.originalHandler.ServeHTTP(resp, req)
}

// ApplyCors wraps original handler by adding cors headers to response BEFORE original ServeHTTP method is called
func ApplyCors(original http.Handler) http.Handler {
	return corsHandler{original}
}

func allowAllCorsActions(resp http.ResponseWriter) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

func isPreflightCorsRequest(req *http.Request) bool {
	isOptionsMethod := req.Method == http.MethodOptions
	containsAccessControlRequestMethod := req.Header.Get("Access-Control-Request-Method") != ""
	containsOriginHeader := req.Header.Get("Origin") != ""
	return isOptionsMethod && containsOriginHeader && containsAccessControlRequestMethod
}

func generatePreflightResponse(req *http.Request, resp http.ResponseWriter) {
	allowAllCorsActions(resp)
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
	cc.originalHandler.ServeHTTP(resp, req)
	resp.Header().Add(cacheControlHeader, noCache)
	resp.Header().Add(cacheControlHeader, noStore)
	resp.Header().Add(cacheControlHeader, mustRevalidate)
}

// DisableCaching middleware adds cache disabling headers to http response
func DisableCaching(original http.Handler) http.Handler {
	return &cacheControl{
		original,
	}
}
