package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

type proposalsRes struct {
	Proposals []proposalRes `json:"proposals"`
}

type locationRes struct {
	ASN     string `json:"asn"`
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
}
type serviceDefinitionRes struct {
	LocationOriginate locationRes `json:"locationOriginate"`
}

type proposalRes struct {
	ID                int                  `json:"id"`
	ProviderId        string               `json:"providerId"`
	ServiceType       string               `json:"serviceType"`
	ServiceDefinition serviceDefinitionRes `json:"serviceDefinition"`
}

func proposalToRes(p dto_discovery.ServiceProposal) proposalRes {
	return proposalRes{
		ID:          p.ID,
		ProviderId:  p.ProviderID,
		ServiceType: p.ServiceType,
		ServiceDefinition: serviceDefinitionRes{
			LocationOriginate: locationRes{
				ASN:     p.ServiceDefinition.GetLocation().ASN,
				Country: p.ServiceDefinition.GetLocation().Country,
				City:    p.ServiceDefinition.GetLocation().City,
			},
		},
	}
}

func mapProposalsToRes(
	proposalArry []dto_discovery.ServiceProposal,
	f func(dto_discovery.ServiceProposal) proposalRes,
) []proposalRes {
	proposalsResArry := make([]proposalRes, len(proposalArry))
	for i, proposal := range proposalArry {
		proposalsResArry[i] = f(proposal)
	}
	return proposalsResArry
}

type proposalsEndpoint struct {
	mysteriumClient server.Client
}

func NewProposalsEndpoint(mc server.Client) *proposalsEndpoint {
	return &proposalsEndpoint{mc}
}

func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {

	providerId := req.URL.Query().Get("providerId")
	proposals, err := pe.mysteriumClient.FindProposals(providerId)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	proposalsRes := proposalsRes{mapProposalsToRes(proposals, proposalToRes)}
	utils.WriteAsJSON(proposalsRes, resp)
}

func AddRoutesForProposals(router *httprouter.Router, mc server.Client) {
	pe := NewProposalsEndpoint(mc)
	router.GET("/proposals", pe.List)
}
