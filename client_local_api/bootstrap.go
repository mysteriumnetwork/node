package client_local_api

import (
	"github.com/mysterium/node/client_local_api/handlers"
	"net/http"
)

func Bootstrap(bindAddress string) error {

	http.Handle("/healthcheck", handlers.HealthCheckHandler)

	return http.ListenAndServe(bindAddress, nil)
}
