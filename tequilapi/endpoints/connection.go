package endpoints

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/service_discovery/dto"
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
	SessionId string `json:"sessionId,omitempty"`
}

type connectionEndpoint struct {
	manager client_connection.Manager
}

func NewConnectionEndpoint(manager client_connection.Manager) *connectionEndpoint {
	return &connectionEndpoint{manager}
}

func (ce *connectionEndpoint) Status(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	statusResponse := toStatusResponse(ce.manager.Status())
	utils.WriteAsJson(statusResponse, resp)
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

	err = ce.manager.Connect(dto.Identity(cr.Identity), cr.NodeKey)

	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
	}
	resp.WriteHeader(http.StatusCreated)
	ce.Status(resp, req, params)
}

func (ce *connectionEndpoint) Kill(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	ce.manager.Disconnect()
	resp.WriteHeader(http.StatusAccepted)
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

func toStatusResponse(status client_connection.Status) statusResponse {
	return statusResponse{
		Status:    fmt.Sprint(status.State),
		SessionId: status.SessionId,
	}
}
