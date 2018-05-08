package endpoints

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client/connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/tequilapi/validation"
	"github.com/mysterium/node/location"
	"net/http"
)

// statusConnectCancelled indicates that connect request was cancelled by user. Since there is no such concept in REST
// operations, custom client error code is defined. Maybe in later times a better idea will come how to handle these situations
const statusConnectCancelled = 499

type connectionRequest struct {
	ConsumerID string `json:"consumerId"`
	ProviderID string `json:"providerId"`
}

type statusResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"sessionId,omitempty"`
}

// ConnectionEndpoint struct represents /connection resource and it's subresources
type ConnectionEndpoint struct {
	manager     connection.Manager
	ipResolver  ip.Resolver
	locationDetector location.Detector
	statsKeeper bytescount.SessionStatsKeeper
	originalLocation location.Location
}

const connectionLogPrefix = "[Connection] "

// NewConnectionEndpoint creates and returns connection endpoint
func NewConnectionEndpoint(manager connection.Manager, ipResolver ip.Resolver, statsKeeper bytescount.SessionStatsKeeper,
	locationDetector location.Detector, originalLocation location.Location) *ConnectionEndpoint {
	return &ConnectionEndpoint{
		manager:     manager,
		ipResolver:  ipResolver,
		statsKeeper: statsKeeper,
		locationDetector: locationDetector,
		originalLocation: originalLocation,
	}
}

// Status returns status of connection
func (ce *ConnectionEndpoint) Status(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	statusResponse := toStatusResponse(ce.manager.Status())
	utils.WriteAsJSON(statusResponse, resp)
}

// Create starts connection
func (ce *ConnectionEndpoint) Create(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	cr, err := toConnectionRequest(req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	errorMap := validateConnectionRequest(cr)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	err = ce.manager.Connect(identity.FromAddress(cr.ConsumerID), identity.FromAddress(cr.ProviderID))

	if err != nil {
		switch err {
		case connection.ErrAlreadyExists:
			utils.SendError(resp, err, http.StatusConflict)
		case connection.ErrConnectionCancelled:
			utils.SendError(resp, err, statusConnectCancelled)
		default:
			log.Error(connectionLogPrefix, err)
			utils.SendError(resp, err, http.StatusInternalServerError)
		}
		return
	}
	resp.WriteHeader(http.StatusCreated)
	ce.Status(resp, req, params)
}

// Kill stops connection
func (ce *ConnectionEndpoint) Kill(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	err := ce.manager.Disconnect()
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			utils.SendError(resp, err, http.StatusConflict)
		default:
			utils.SendError(resp, err, http.StatusInternalServerError)
		}
		return
	}
	resp.WriteHeader(http.StatusAccepted)
}

// GetIP responds with current ip, using its ip resolver
func (ce *ConnectionEndpoint) GetIP(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ip, err := ce.ipResolver.GetPublicIP()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}
	response := struct {
		IP string `json:"ip"`
	}{
		IP: ip,
	}
	utils.WriteAsJSON(response, writer)
}

// GetLocation responds with original and current countries
func (ce *ConnectionEndpoint) GetLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	currentLocation, err := ce.locationDetector.DetectLocation()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	response := struct {
		Original location.Location `json:"original"`
		Current location.Location `json:"current"`
	}{
		Original: ce.originalLocation,
		Current: currentLocation,
	}
	utils.WriteAsJSON(response, writer)
}

// GetStatistics returns statistics about current connection
func (ce *ConnectionEndpoint) GetStatistics(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	stats := ce.statsKeeper.Retrieve()

	duration := ce.statsKeeper.GetSessionDuration()

	response := struct {
		BytesSent     int `json:"bytesSent"`
		BytesReceived int `json:"bytesReceived"`
		Duration      int `json:"duration"`
	}{
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
		Duration:      int(duration.Seconds()),
	}
	utils.WriteAsJSON(response, writer)
}

// AddRoutesForConnection adds connections routes to given router
func AddRoutesForConnection(router *httprouter.Router, manager connection.Manager, ipResolver ip.Resolver,
	statsKeeper bytescount.SessionStatsKeeper, locationDetector location.Detector, originalLocation location.Location) {
	connectionEndpoint := NewConnectionEndpoint(manager, ipResolver, statsKeeper, locationDetector, originalLocation)
	router.GET("/connection", connectionEndpoint.Status)
	router.PUT("/connection", connectionEndpoint.Create)
	router.DELETE("/connection", connectionEndpoint.Kill)
	router.GET("/connection/ip", connectionEndpoint.GetIP)
	router.GET("/connection/statistics", connectionEndpoint.GetStatistics)
	router.GET("/connection/location", connectionEndpoint.GetLocation)
}

func toConnectionRequest(req *http.Request) (*connectionRequest, error) {
	var connectionRequest = connectionRequest{}
	err := json.NewDecoder(req.Body).Decode(&connectionRequest)
	if err != nil {
		return nil, err
	}
	return &connectionRequest, nil
}

func validateConnectionRequest(cr *connectionRequest) *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if len(cr.ConsumerID) == 0 {
		errors.ForField("consumerId").AddError("required", "Field is required")
	}
	if len(cr.ProviderID) == 0 {
		errors.ForField("providerId").AddError("required", "Field is required")
	}
	return errors
}

func toStatusResponse(status connection.ConnectionStatus) statusResponse {
	return statusResponse{
		Status:    string(status.State),
		SessionID: string(status.SessionID),
	}
}
