/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/metrics"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model ProposalsList
type proposalsRes struct {
	Proposals []proposalRes `json:"proposals"`
}

// swagger:model ServiceLocationDTO
type locationRes struct {
	// example: EU
	Continent string `json:"continent,omitempty"`
	// example: NL
	Country string `json:"country,omitempty"`
	// example: Amsterdam
	City string `json:"city,omitempty"`

	// Autonomous System Number
	// example: 00001
	ASN int `json:"asn"`
	// example: Telia Lietuva, AB
	ISP string `json:"isp,omitempty"`
	// example: residential
	NodeType string `json:"node_type,omitempty"`
}

// swagger:model ServiceDefinitionDTO
type serviceDefinitionRes struct {
	LocationOriginate locationRes `json:"locationOriginate"`
}

// swagger:model ProposalDTO
type proposalRes struct {
	// per provider unique serial number of service description provided
	// example: 5
	ID int `json:"id"`

	// provider who offers service
	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"providerId"`

	// type of service provider offers
	// example: openvpn
	ServiceType string `json:"serviceType"`

	// qualitative service definition
	ServiceDefinition serviceDefinitionRes `json:"serviceDefinition"`

	// Metrics of the service
	Metrics json.RawMessage `json:"metrics,omitempty"`

	// ACL
	ACL *[]market.ACL `json:"acl,omitempty"`
}

func proposalToRes(p market.ServiceProposal) proposalRes {
	return proposalRes{
		ID:          p.ID,
		ProviderID:  p.ProviderID,
		ServiceType: p.ServiceType,
		ServiceDefinition: serviceDefinitionRes{
			LocationOriginate: locationRes{
				Continent: p.ServiceDefinition.GetLocation().Continent,
				Country:   p.ServiceDefinition.GetLocation().Country,
				City:      p.ServiceDefinition.GetLocation().City,

				ASN:      p.ServiceDefinition.GetLocation().ASN,
				ISP:      p.ServiceDefinition.GetLocation().ISP,
				NodeType: p.ServiceDefinition.GetLocation().NodeType,
			},
		},
		ACL: p.ACL,
	}
}

func mapProposalsToRes(
	proposalArry []market.ServiceProposal,
	f func(market.ServiceProposal) proposalRes,
	metrics func(proposalRes) proposalRes,
) []proposalRes {
	proposalsResArry := make([]proposalRes, len(proposalArry))
	for i, proposal := range proposalArry {
		proposalsResArry[i] = metrics(f(proposal))
	}
	return proposalsResArry
}

// ProposalProvider allows to fetch proposals by specified params
type ProposalProvider interface {
	FindProposals(providerID string, serviceType string) ([]market.ServiceProposal, error)
}

type proposalsEndpoint struct {
	proposalProvider     ProposalProvider
	mysteriumMorqaClient metrics.QualityOracle
}

// NewProposalsEndpoint creates and returns proposal creation endpoint
func NewProposalsEndpoint(proposalProvider ProposalProvider, morqaClient metrics.QualityOracle) *proposalsEndpoint {
	return &proposalsEndpoint{proposalProvider, morqaClient}
}

// swagger:operation GET /proposals Proposal listProposals
// ---
// summary: Returns proposals
// description: Returns list of proposals filtered by provider id
// parameters:
//   - in: query
//     name: providerId
//     description: id of provider proposals
//     type: string
//   - in: query
//     name: serviceType
//     description: the service type of the proposal. Possible values are "openvpn", "wireguard" and "noop"
//     type: string
//   - in: query
//     name: fetchConnectCounts
//     description: if set to true, fetches the connection success metrics for nodes. False by default.
//     type: boolean
// responses:
//   200:
//     description: List of proposals
//     schema:
//       "$ref": "#/definitions/ProposalsList"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	providerID := req.URL.Query().Get("providerId")
	serviceType := req.URL.Query().Get("serviceType")
	fetchConnectCounts := req.URL.Query().Get("fetchConnectCounts")
	proposals, err := pe.proposalProvider.FindProposals(providerID, serviceType)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	addMetricsToRes := noMetrics
	if fetchConnectCounts == "true" {
		addMetricsToRes = addMetrics(pe.mysteriumMorqaClient)
	}

	proposalsRes := proposalsRes{mapProposalsToRes(proposals, proposalToRes, addMetricsToRes)}
	utils.WriteAsJSON(proposalsRes, resp)
}

// AddRoutesForProposals attaches proposals endpoints to router
func AddRoutesForProposals(router *httprouter.Router, proposalProvider ProposalProvider, morqaClient metrics.QualityOracle) {
	pe := NewProposalsEndpoint(proposalProvider, morqaClient)
	router.GET("/proposals", pe.List)
}

func noMetrics(p proposalRes) proposalRes { return p }

func addMetrics(mc metrics.QualityOracle) func(p proposalRes) proposalRes {
	receivedMetrics := mc.ProposalsMetrics()
	proposalsMetrics := make(map[string]json.RawMessage, len(receivedMetrics))
	var proposal struct{ ProposalID proposalRes }

	for _, m := range receivedMetrics {
		json, err := metrics.Parse(m, &proposal)
		if err != nil {
			return noMetrics
		}
		p := proposal.ProposalID
		proposalsMetrics[p.ProviderID+"-"+p.ServiceType] = json
	}

	return func(p proposalRes) proposalRes {
		if metrics, ok := proposalsMetrics[p.ProviderID+"-"+p.ServiceType]; ok {
			p.Metrics = metrics
			return p
		}
		p.Metrics = []byte("{}")
		return p
	}
}
