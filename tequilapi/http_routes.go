package tequilapi

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/endpoints"
	"time"
)

func NewApiEndpoints() *httprouter.Router {
	router := httprouter.New()
	router.HandleMethodNotAllowed = true

	router.GET("/healthcheck", endpoints.HealthCheckEndpointFactory(time.Now).HealthCheck)
	return router
}
