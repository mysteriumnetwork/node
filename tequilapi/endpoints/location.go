package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
	"github.com/mysterium/node/client/connection"
)

// LocationEndpoint struct represents /location resource and it's subresources
type LocationEndpoint struct {
	manager connection.Manager
	locationDetector location.Detector
	originalLocationCache location.Cache
}

// NewLocationEndpoint creates and returns location endpoint
func NewLocationEndpoint(manager connection.Manager, locationDetector location.Detector,
	originalLocationCache location.Cache) *LocationEndpoint {
	return &LocationEndpoint{
		manager: manager,
		locationDetector: locationDetector,
		originalLocationCache: originalLocationCache,
	}
}

// GetLocation responds with original and current countries
func (ce *LocationEndpoint) GetLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	originalLocation, err := ce.originalLocationCache.Get()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	var currentLocation location.Location

	if ce.manager.Status().State == connection.Connected {
		_currentLocation, err := ce.locationDetector.DetectLocation()
		currentLocation = _currentLocation
		if err != nil {
			utils.SendError(writer, err, http.StatusServiceUnavailable)
			return
		}
	}

	response := struct {
		Original location.Location `json:"original"`
		Current location.Location `json:"current"`
	}{
		Original: originalLocation,
		Current: currentLocation,
	}
	utils.WriteAsJSON(response, writer)
}

// AddRoutesForLocation adds location routes to given router
func AddRoutesForLocation(router *httprouter.Router, manager connection.Manager,
	locationDetector location.Detector, locationCache location.Cache) {

	locationEndpoint := NewLocationEndpoint(manager, locationDetector, locationCache)
	router.GET("/location", locationEndpoint.GetLocation)
}
