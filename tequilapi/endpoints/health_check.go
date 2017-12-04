package endpoints

import (
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
	"time"
)

type healthCheckData struct {
	Uptime string `json:"uptime"`
}

/*
HealthCheckEndpointFactory creates http.HandlerFunc for injected current time provider,
which returns health status json object, currently with only one field - uptime
*/
func HealthCheckEndpointFactory(currentTime func() time.Time) http.HandlerFunc {
	startupTime := currentTime()
	return func(writer http.ResponseWriter, request *http.Request) {
		status := healthCheckData{
			Uptime: currentTime().Sub(startupTime).String(),
		}
		utils.WriteAsJson(status, writer)
	}
}
