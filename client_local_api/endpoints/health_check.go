package endpoints

import (
	"github.com/mysterium/node/client_local_api/utils"
	"net/http"
	"time"
)

var currentTime = time.Now

var startupTime = currentTime()

type healthCheckData struct {
	Uptime string `json:"uptime"`
}

/*
HealthCheckEndpoint function returns health status json object, currently with only one field - uptime
*/
func HealthCheckEndpoint(writer http.ResponseWriter, request *http.Request) {
	status := healthCheckData{
		Uptime: currentTime().Sub(startupTime).String(),
	}
	utils.WriteAsJson(status, writer)
}
