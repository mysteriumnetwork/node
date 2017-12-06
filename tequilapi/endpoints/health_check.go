package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
	"time"
)

type healthCheckData struct {
	Uptime string `json:"uptime"`
}

type healthCheckEndpoint struct {
	startTime       time.Time
	currentTimeFunc func() time.Time
}

/*
HealthCheckEndpointFactory creates a structure with single HealthCheck method for healthcheck serving as http,
currentTimeFunc is injected for easier testing
*/
func HealthCheckEndpointFactory(currentTimeFunc func() time.Time) *healthCheckEndpoint {
	startTime := currentTimeFunc()
	return &healthCheckEndpoint{startTime, currentTimeFunc}
}

func (hce *healthCheckEndpoint) HealthCheck(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	status := healthCheckData{
		Uptime: hce.currentTimeFunc().Sub(hce.startTime).String(),
	}
	utils.WriteAsJson(status, writer)
}
