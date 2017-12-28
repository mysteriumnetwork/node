package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

type proposalsEndpoint struct {
	mysteriumClient server.Client
}

type proposalsDto struct {
	Proposals []dto_discovery.ServiceProposal `json:"proposals"`
}

func NewProposalsEndpoint(mc server.Client) *proposalsEndpoint {
	return &proposalsEndpoint{mc}
}

func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {

	nodeKey := params.ByName("nodeKey")
	proposals, err := pe.mysteriumClient.FindProposals(nodeKey)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	proposalsDto := proposalsDto{proposals}
	utils.WriteAsJson(proposalsDto, resp)
}

func AddRoutesForProposals(router *httprouter.Router, mc server.Client) {
	pe := NewProposalsEndpoint(mc)
	router.GET("/proposals", pe.List)
}
