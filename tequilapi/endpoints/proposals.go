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
	ProviderID        string               `json:"providerId"`
	ServiceType       string               `json:"serviceType"`
	ServiceDefinition serviceDefinitionRes `json:"serviceDefinition"`
}

func proposalToRes(p dto_discovery.ServiceProposal) proposalRes {
	return proposalRes{
		ID:          p.ID,
		ProviderID:  p.ProviderID,
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

// NewProposalsEndpoint creates and returns proposal creation endpoint
func NewProposalsEndpoint(mc server.Client) *proposalsEndpoint {
	return &proposalsEndpoint{mc}
}

// List returns list of proposals
func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {

	providerID := req.URL.Query().Get("providerId")
	proposals, err := pe.mysteriumClient.FindProposals(providerID)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	proposalsRes := proposalsRes{mapProposalsToRes(proposals, proposalToRes)}
	utils.WriteAsJSON(proposalsRes, resp)
}

// AddRoutesForProposals attaches proposals endpoints to router
func AddRoutesForProposals(router *httprouter.Router, mc server.Client) {
	pe := NewProposalsEndpoint(mc)
	router.GET("/proposals", pe.List)
}
