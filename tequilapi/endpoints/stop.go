package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// Stopper stops application and performs required cleanup tasks
type Stopper func()

// AddRouteForStop adds stop route to given router
func AddRouteForStop(router *httprouter.Router, stop Stopper) {
	router.POST("/stop", newStopHandler(stop))
}

func newStopHandler(stop Stopper) httprouter.Handle {
	return func(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		stop()
	}
}
