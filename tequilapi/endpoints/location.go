/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client/connection"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

// LocationEndpoint struct represents /location resource and it's subresources
type LocationEndpoint struct {
	manager               connection.Manager
	locationDetector      location.Detector
	originalLocationCache location.Cache
}

// NewLocationEndpoint creates and returns location endpoint
func NewLocationEndpoint(manager connection.Manager, locationDetector location.Detector,
	originalLocationCache location.Cache) *LocationEndpoint {
	return &LocationEndpoint{
		manager:               manager,
		locationDetector:      locationDetector,
		originalLocationCache: originalLocationCache,
	}
}

// GetLocation responds with original and current countries
func (le *LocationEndpoint) GetLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	originalLocation := le.originalLocationCache.Get()

	var currentLocation location.Location
	var err error
	if le.manager.Status().State == connection.Connected {
		currentLocation, err = le.locationDetector.DetectLocation()
		if err != nil {
			utils.SendError(writer, err, http.StatusServiceUnavailable)
			return
		}
	} else {
		currentLocation = originalLocation
	}

	response := struct {
		Original location.Location `json:"original"`
		Current  location.Location `json:"current"`
	}{
		Original: originalLocation,
		Current:  currentLocation,
	}
	utils.WriteAsJSON(response, writer)
}

// AddRoutesForLocation adds location routes to given router
func AddRoutesForLocation(router *httprouter.Router, manager connection.Manager,
	locationDetector location.Detector, locationCache location.Cache) {

	locationEndpoint := NewLocationEndpoint(manager, locationDetector, locationCache)
	router.GET("/location", locationEndpoint.GetLocation)
}
