package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

// LocationEndpoint struct represents /location resource and it's subresources
type LocationEndpoint struct {
	locationDetector location.Detector
	locationCache location.Cache
}

// NewLocationEndpoint creates and returns location endpoint
func NewLocationEndpoint(locationDetector location.Detector, locationCache location.Cache) *LocationEndpoint {
	return &LocationEndpoint{
		locationDetector: locationDetector,
		locationCache: locationCache,
	}
}

// GetLocation responds with original and current countries
func (ce *LocationEndpoint) GetLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	originalLocation, err := ce.locationCache.Get()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	currentLocation, err := ce.locationDetector.DetectLocation()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
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
func AddRoutesForLocation(router *httprouter.Router, locationDetector location.Detector, locationCache location.Cache) {
	locationEndpoint := NewLocationEndpoint(locationDetector, locationCache)
	router.GET("/location", locationEndpoint.GetLocation)
}
