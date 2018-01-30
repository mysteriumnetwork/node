package endpoints

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/tequilapi/validation"
	"net/http"
)

type connectionRequest struct {
	Identity string `json:"identity"`
	NodeKey  string `json:"nodeKey"`
}

type statusResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"sessionId,omitempty"`
}

type connectionEndpoint struct {
	manager     client_connection.Manager
	ipResolver  ip.Resolver
	statsKeeper bytescount.SessionStatsKeeper
}

func NewConnectionEndpoint(manager client_connection.Manager, ipResolver ip.Resolver, statsKeeper bytescount.SessionStatsKeeper) *connectionEndpoint {
	return &connectionEndpoint{
		manager:     manager,
		ipResolver:  ipResolver,
		statsKeeper: statsKeeper,
	}
}

func (ce *connectionEndpoint) Status(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	statusResponse := toStatusResponse(ce.manager.Status())
	utils.WriteAsJSON(statusResponse, resp)
}

func (ce *connectionEndpoint) Create(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
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

	err = ce.manager.Connect(identity.FromAddress(cr.Identity), cr.NodeKey)

	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(http.StatusCreated)
	ce.Status(resp, req, params)
}

func (ce *connectionEndpoint) Kill(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	ce.manager.Disconnect()
	resp.WriteHeader(http.StatusAccepted)
}

// GetIP responds with current ip, using its ip resolver
func (ce *connectionEndpoint) GetIP(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ip, err := ce.ipResolver.GetPublicIP()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}
	response := struct {
		IP string `json:"ip"`
	}{
		IP: ip,
	}
	utils.WriteAsJSON(response, writer)
}

// GetStatistics returns statistics about current connection
func (ce *connectionEndpoint) GetStatistics(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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
func AddRoutesForConnection(router *httprouter.Router, manager client_connection.Manager, ipResolver ip.Resolver,
	statsKeeper bytescount.SessionStatsKeeper) {
	connectionEndpoint := NewConnectionEndpoint(manager, ipResolver, statsKeeper)
	router.GET("/connection", connectionEndpoint.Status)
	router.PUT("/connection", connectionEndpoint.Create)
	router.DELETE("/connection", connectionEndpoint.Kill)
	router.GET("/connection/ip", connectionEndpoint.GetIP)
	router.GET("/connection/statistics", connectionEndpoint.GetStatistics)
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
	if len(cr.Identity) == 0 {
		errors.ForField("identity").AddError("required", "Field is required")
	}
	if len(cr.NodeKey) == 0 {
		errors.ForField("nodeKey").AddError("required", "Field is required")
	}
	return errors
}

func toStatusResponse(status client_connection.ConnectionStatus) statusResponse {
	return statusResponse{
		Status:    string(status.State),
		SessionID: string(status.SessionID),
	}
}
