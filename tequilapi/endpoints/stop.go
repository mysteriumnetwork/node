/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// ApplicationStopper stops application and performs required cleanup tasks
type ApplicationStopper func()

// AddRouteForStop adds stop route to given router
func AddRouteForStop(router *httprouter.Router, stop ApplicationStopper) {
	router.POST("/stop", newStopHandler(stop))
}

// swagger:operation POST /stop Client applicationStop
// ---
// summary: Stops client
// description: Initiates client termination
// responses:
//   202:
//     description: Request accepted, stopping
func newStopHandler(stop ApplicationStopper) httprouter.Handle {
	return func(response http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		log.Info("Application stop requested")

		go callStopWhenNotified(req.Context().Done(), stop)
		response.WriteHeader(http.StatusAccepted)
	}
}
func callStopWhenNotified(notify <-chan struct{}, stopApplication ApplicationStopper) {
	<-notify
	stopApplication()
}
