package tequilapi

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/tequilapi/endpoints"
	"os"
	"time"
)

// NewAPIRouter returns new api router with status endpoints
func NewAPIRouter() *httprouter.Router {
	router := httprouter.New()
	router.HandleMethodNotAllowed = true

	router.GET("/healthcheck", endpoints.HealthCheckEndpointFactory(time.Now, os.Getpid).HealthCheck)

	ipifyClient := ipify.NewClient()
	router.GET("/ip", endpoints.NewCurrentIPEndpoint(ipifyClient.GetPublicIP).CurrentIP)

	return router
}
