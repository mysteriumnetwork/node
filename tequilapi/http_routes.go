package tequilapi

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/tequilapi/endpoints"
	"time"
)

func NewApiEndpoints(idm identity.IdentityManagerInterface) *httprouter.Router {
	router := httprouter.New()
	router.HandleMethodNotAllowed = true

	router.GET("/healthcheck", endpoints.HealthCheckEndpointFactory(time.Now).HealthCheck)
	router.GET("/identities", endpoints.NewIdentitiesEndpoint(idm).List)
	return router
}
