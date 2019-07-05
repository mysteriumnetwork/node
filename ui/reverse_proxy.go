package ui

import (
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/tequilapi/endpoints"
)

func buildTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 20 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     15,
	}
}

func buildReverseProxy(transport *http.Transport, tequilapiPort int) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = tequilapiHost + ":" + strconv.Itoa(tequilapiPort)
			req.URL.Path = strings.Replace(req.URL.Path, tequilapiUrlPrefix, "", 1)
			req.URL.Path = strings.TrimRight(req.URL.Path, "/")
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Set("Access-Control-Allow-Origin", "*")
			res.Header.Set("Access-Control-Allow-Headers", "*")
			res.Header.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			return nil
		},
		Transport: transport,
	}

	proxy.FlushInterval = 10 * time.Millisecond

	return proxy
}

// ReverseTequilapiProxy proxies UIServer requests to the TequilAPI server
func ReverseTequilapiProxy(tequilapiPort int, authenticator jwtAuthenticator) gin.HandlerFunc {
	proxy := buildReverseProxy(buildTransport(), tequilapiPort)

	return func(c *gin.Context) {
		// skip non Tequilapi routes
		if !isTequilapiURL(c.Request.URL.Path) {
			return
		}

		// authenticate all but the login route
		if !isTequilapiURL(c.Request.URL.Path, endpoints.TequilapiLoginEndpointPath) {
			cookieToken, err := c.Cookie(auth.JWTCookieName)

			if err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			if _, err := authenticator.ValidateToken(cookieToken); err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}

		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					// ignore streaming errors (SSE)
					// there's nothing we can do about them
				} else {
					panic(err)
				}
			}
		}()

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func isTequilapiURL(url string, endpoints ...string) bool {
	return strings.Contains(url, tequilapiUrlPrefix+strings.Join(endpoints, ""))
}
