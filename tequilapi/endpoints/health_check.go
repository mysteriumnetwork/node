package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/version"
	"net/http"
	"time"
)

type healthCheckData struct {
	Uptime  string    `json:"uptime"`
	Process int       `json:"process"`
	Version buildInfo `json:"version"`
}

type buildInfo struct {
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
	BuildNumber string `json:"buildNumber"`
}

type healthCheckEndpoint struct {
	startTime       time.Time
	currentTimeFunc func() time.Time
	processNumber   int
	versionInfo     *version.Info
}

/*
HealthCheckEndpointFactory creates a structure with single HealthCheck method for healthcheck serving as http,
currentTimeFunc is injected for easier testing
*/
func HealthCheckEndpointFactory(currentTimeFunc func() time.Time, procID func() int, versionInfo *version.Info) *healthCheckEndpoint {
	startTime := currentTimeFunc()
	return &healthCheckEndpoint{
		startTime,
		currentTimeFunc,
		procID(),
		versionInfo,
	}
}

func (hce *healthCheckEndpoint) HealthCheck(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	status := healthCheckData{
		Uptime:  hce.currentTimeFunc().Sub(hce.startTime).String(),
		Process: hce.processNumber,
		Version: buildInfo{
			hce.versionInfo.Commit,
			hce.versionInfo.Branch,
			hce.versionInfo.BuildNumber,
		},
	}
	utils.WriteAsJSON(status, writer)
}
