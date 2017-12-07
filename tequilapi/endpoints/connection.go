package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

type connectionRequest struct {
	Identity string `json:"identity"`
}

type connectionStatus struct {
	Status string `json:"status"`
}

type connectionEndpoint struct {
}

func NewConnectionEndpoint() *connectionEndpoint {
	return &connectionEndpoint{}
}

func (ce *connectionEndpoint) Status(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp.WriteHeader(http.StatusNotFound)
}

func (ce *connectionEndpoint) Create(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	utils.WriteAsJson(connectionStatus{"Viskas ok!"}, resp)
}

func (ce *connectionEndpoint) Kill(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp.WriteHeader(http.StatusAccepted)
}
