package tequil

import "strings"

const TequilapiURLPrefix = "/tequilapi"

var UnprotectedRoutes = []string{"/auth/authenticate", "/auth/login", "/healthcheck"}

func IsUnprotectedRoute(url string) bool {
	for _, route := range UnprotectedRoutes {
		if strings.Contains(url, route) {
			return true
		}
	}

	return false
}

func IsProtectedRoute(url string) bool {
	return !IsUnprotectedRoute(url)
}

func IsReverseProxyRoute(url string) bool {
	return strings.Contains(url, TequilapiURLPrefix)
}
