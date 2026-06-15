package tequil

import "strings"

const TequilapiURLPrefix = "/tequilapi"

var UnprotectedRoutes = []string{"/auth/authenticate", "/auth/login", "/healthcheck", "/config/ui/features"}

func IsUnprotectedRoute(url string) bool {
	for _, route := range UnprotectedRoutes {
		if url == route || strings.HasPrefix(url, route+"/") || strings.HasPrefix(url, route+"?") {
			return true
		}
	}
	return false
}

func IsProtectedRoute(url string) bool {
	return !IsUnprotectedRoute(url)
}

func IsReverseProxyRoute(url string) bool {
	return strings.HasPrefix(url, TequilapiURLPrefix) || strings.Contains(url, TequilapiURLPrefix)
}
