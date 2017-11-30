package client_local_api

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/client_local_api/endpoints"
	"net/http"
	"reflect"
	"runtime"
)

const httpLogPrefix = "[http]"

func init() {
	RegisterEndpoint("/healthcheck", endpoints.HealthCheckEndpoint)
}

/*
Bootstrap function starts http server on specified binding address, which format conforms to what is expeted by
http.ListenAndServe function
*/
func Bootstrap(bindAddress string) {
	log.Infof("%s Local api binding %s\n", httpLogPrefix, bindAddress)
	log.Errorf(httpLogPrefix, http.ListenAndServe(bindAddress, nil))
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
