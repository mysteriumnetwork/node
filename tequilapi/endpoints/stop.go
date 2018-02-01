package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type Stopper func()

func AddRouteForStop(router *httprouter.Router, stop Stopper) {
	router.POST("/stop", newStopHandler(stop))
}

func newStopHandler(stop Stopper) httprouter.Handle {
	return func(response http.ResponseWriter, request *http.Request, params httprouter.Params) {
		stop()
	}
}
