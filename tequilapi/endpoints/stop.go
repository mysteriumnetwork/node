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

	"github.com/gin-gonic/gin"

	"github.com/rs/zerolog/log"
)

// ApplicationStopper stops application and performs required cleanup tasks
type ApplicationStopper func()

// AddRouteForStop adds stop route to given router
func AddRouteForStop(stop ApplicationStopper) func(*gin.Engine) error {
	return func(e *gin.Engine) error {
		e.POST("/stop", newStopHandler(stop))
		return nil
	}
}

// swagger:operation POST /stop Client applicationStop
//
//	---
//	summary: Stops client
//	description: Initiates client termination
//	responses:
//	  202:
//	    description: Request accepted, stopping
func newStopHandler(stop ApplicationStopper) func(*gin.Context) {
	return func(c *gin.Context) {
		log.Info().Msg("Application stop requested")

		go callStopWhenNotified(c.Request.Context().Done(), stop)
		c.Status(http.StatusAccepted)
	}
}
func callStopWhenNotified(notify <-chan struct{}, stopApplication ApplicationStopper) {
	<-notify
	stopApplication()
}
