package endpoints

import (
	log "github.com/cihub/seelog"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// ApplicationStopper stops application and performs required cleanup tasks
type ApplicationStopper func()

// AddRouteForStop adds stop route to given router
func AddRouteForStop(router *httprouter.Router, stop ApplicationStopper) {
	router.POST("/stop", newStopHandler(stop))
}

func newStopHandler(stop ApplicationStopper) httprouter.Handle {
	return func(response http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		log.Info("Application stop requested")

		go doWhenNotified(req.Context().Done(), stop)
		response.WriteHeader(http.StatusAccepted)
	}
}
func doWhenNotified(notify <-chan struct{}, stopApplication ApplicationStopper) {
	<-notify
	stopApplication()
}
