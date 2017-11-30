package client_local_api

import (
	"fmt"
	"github.com/mysterium/node/client_local_api/endpoints"
	_ "github.com/mysterium/node/client_local_api/endpoints"
	"net/http"
)

func init() {
	RegisterEndpoint("/healthcheck", endpoints.HealthCheckHandler)
}

/*
Bootstrap function starts http server on specified binding address, which format conforms to what is expeted by
http.ListenAndServe function
*/
func Bootstrap(bindAddress string) error {
	fmt.Println("Binding local http server on: ", bindAddress)
	return http.ListenAndServe(bindAddress, nil)
}

/*
RegisterEndpoint registers http.HandlerFunc to specified path. It depends on http.Handle function and simply provides
single point for endpoint registration, with posibility to add custom global handlers like enforcing utf-8 responses
*/
func RegisterEndpoint(path string, handler http.HandlerFunc) {
	fmt.Println("Registering path: ", path)
	http.Handle(path, handler)
}
