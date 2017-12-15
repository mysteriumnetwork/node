package tequilapi

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/tequilapi/endpoints"
	"time"
)

func NewApiEndpoints(idm identity.IdentityManagerInterface, connManager client_connection.Manager) *httprouter.Router {
	router := httprouter.New()
	router.HandleMethodNotAllowed = true

	router.GET("/healthcheck", endpoints.HealthCheckEndpointFactory(time.Now).HealthCheck)

	router.GET("/identities", endpoints.NewIdentitiesEndpoint(idm).List)

	connectionEndpoint := endpoints.NewConnectionEndpoint(connManager)
	router.GET("/connection", connectionEndpoint.Status)
	router.PUT("/connection", connectionEndpoint.Create)
	router.DELETE("/connection", connectionEndpoint.Kill)

	return router
}
