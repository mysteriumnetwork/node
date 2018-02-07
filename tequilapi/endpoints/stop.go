package endpoints

import (
	log "github.com/cihub/seelog"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"time"
)

// Stopper stops application and performs required cleanup tasks
type Stopper func()

// AddRouteForStop adds stop route to given router
func AddRouteForStop(router *httprouter.Router, stop Stopper, delay time.Duration) {
	router.POST("/stop", newStopHandler(stop, delay))
}

func newStopHandler(stop Stopper, delay time.Duration) httprouter.Handle {
	return func(response http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		log.Info("Client is stopping")
		delayExecution(stop, delay)
		response.WriteHeader(http.StatusAccepted)
	}
}

func delayExecution(function func(), duration time.Duration) {
	go func() {
		time.Sleep(duration)
		function()
	}()
}
