/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/consumer/entertainment"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type entertainmentEndpoint struct {
	estimator estimator
}

type estimator interface {
	EstimatedEntertainment(myst float64) entertainment.Estimates
}

// swagger:operation GET /entertainment Entertainment Estimate
//
//	---
//	summary: Estimate entertainment durations/data cap for the MYST amount specified.
//	description: Estimate entertainment durations/data cap for the MYST amount specified.
//	parameters:
//	- name: amount
//	  in: query
//	  description: Amount of MYST to give entertainment estimates for.
//	  type: integer
//	  required: true
//	responses:
//	  200:
//	    description: Entertainment estimates
//	    schema:
//	      "$ref": "#/definitions/EntertainmentEstimateResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (e *entertainmentEndpoint) Estimate(c *gin.Context) {
	req := contract.EntertainmentEstimateRequest{}
	if err := req.Bind(c.Request); err != nil {
		c.Error(err)
		return
	}

	estimates := e.estimator.EstimatedEntertainment(req.Amount)
	res := contract.EntertainmentEstimateResponse{
		VideoMinutes:    estimates.VideoMinutes,
		MusicMinutes:    estimates.MusicMinutes,
		BrowsingMinutes: estimates.BrowsingMinutes,
		TrafficMB:       estimates.TrafficMB,
		PriceGiB:        estimates.PricePerGiB,
		PriceMin:        estimates.PricePerMin,
	}
	utils.WriteAsJSON(res, c.Writer)
}

// AddEntertainmentRoutes registers routes for entertainment.
func AddEntertainmentRoutes(estimator estimator) func(*gin.Engine) error {
	endpoint := &entertainmentEndpoint{estimator: estimator}
	return func(e *gin.Engine) error {
		g := e.Group("/entertainment")
		g.GET("", endpoint.Estimate)
		return nil
	}
}
