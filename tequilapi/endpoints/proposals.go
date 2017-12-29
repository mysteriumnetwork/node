package endpoints

import (
    "github.com/julienschmidt/httprouter"
    "github.com/mysterium/node/server"
    dto_discovery "github.com/mysterium/node/service_discovery/dto"
    "github.com/mysterium/node/tequilapi/utils"
    "net/http"
)

type proposalsDto struct {
    Proposals []proposalTeqDto `json:"proposals"`
}

type locationTeqDto struct {
    ASN string `json:"asn"`
}
type serviceDefinitionTeqDto struct {
    LocationOriginate locationTeqDto `json:"locationOriginate"`
}

type proposalTeqDto struct {
    Id                int                     `json:"id"`
    ProviderId        string                  `json:"providerId"`
    ServiceDefinition serviceDefinitionTeqDto `json:"serviceDefinition"`
}

func proposalToDto(p dto_discovery.ServiceProposal) proposalTeqDto {
    return proposalTeqDto{
        Id:         p.Id,
        ProviderId: p.ProviderId,
        ServiceDefinition: serviceDefinitionTeqDto{
            LocationOriginate: locationTeqDto{
                ASN: p.ServiceDefinition.GetLocation().ASN,
            },
        },
    }
}

func mapProposalsToTeqDto(
    proposalArry []dto_discovery.ServiceProposal,
    f func(dto_discovery.ServiceProposal) proposalTeqDto,
) []proposalTeqDto {
    proposalsDtoArry := make([]proposalTeqDto, len(proposalArry))
    for i, proposal := range proposalArry {
        proposalsDtoArry[i] = f(proposal)
    }
    return proposalsDtoArry
}

type proposalsEndpoint struct {
    mysteriumClient server.Client
}

func NewProposalsEndpoint(mc server.Client) *proposalsEndpoint {
    return &proposalsEndpoint{mc}
}

func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {

    nodeId := req.URL.Query().Get("provider_id")
    proposals, err := pe.mysteriumClient.FindProposals(nodeId)
    if err != nil {
        utils.SendError(resp, err, http.StatusInternalServerError)
        return
    }
    proposalsDto := proposalsDto{mapProposalsToTeqDto(proposals, proposalToDto)}
    utils.WriteAsJson(proposalsDto, resp)
}

func AddRoutesForProposals(router *httprouter.Router, mc server.Client) {
    pe := NewProposalsEndpoint(mc)
    router.GET("/proposals", pe.List)
}
