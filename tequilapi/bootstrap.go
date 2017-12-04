package tequilapi

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/tequilapi/endpoints"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

const httpLogPrefix = "[http]"

func registerAllEndpoints() {
	RegisterEndpoint("/healthcheck", endpoints.HealthCheckEndpointFactory(time.Now))
}

/*
Bootstrap function starts http server on specified binding address, which format conforms to what is expected by
http.ListenAndServe function. This function IS BLOCKING, that means - it should be run on goroutine to resume with normal flow
*/
func Bootstrap(bindAddress string, bindPort int) (ApiServer, error) {
	registerAllEndpoints()
	apiServer, err := CreateNew(bindAddress, bindPort)
	if err != nil {
		return nil, err
	}
	return apiServer, nil
}

/*
RegisterEndpoint registers http.HandlerFunc to specified path. It depends on http.Handle function and simply provides
single point for endpoint registration, with posibility to add custom global handlers like enforcing utf-8 responses
*/
func RegisterEndpoint(path string, handler http.HandlerFunc) {
	log.Tracef("%s Mapping %s -> %s", httpLogPrefix, path, extractName(handler))
	http.Handle(path, handler)
}

func extractName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
