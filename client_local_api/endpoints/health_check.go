package endpoints

import (
	"net/http"
	"time"

	"github.com/mysterium/node/client_local_api/utils"
)

var currentTime = time.Now

var startupTime = currentTime()

type healthCheckData struct {
	Uptime string `json:"uptime"`
}

var HealthCheckHandler = func(writer http.ResponseWriter, request *http.Request) {
	status := healthCheckData{
		Uptime: currentTime().Sub(startupTime).String(),
	}
	utils.WriteAsJson(status, writer)
}
