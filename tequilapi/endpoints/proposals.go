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
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// QualityFinder allows to fetch proposal quality data
type QualityFinder interface {
	ProposalsMetrics() []quality.ConnectMetric
}

type proposalsEndpoint struct {
	proposalRepository proposal.Repository
	qualityProvider    QualityFinder
}

// NewProposalsEndpoint creates and returns proposal creation endpoint
func NewProposalsEndpoint(proposalRepository proposal.Repository, qualityProvider QualityFinder) *proposalsEndpoint {
	return &proposalsEndpoint{
		proposalRepository: proposalRepository,
		qualityProvider:    qualityProvider,
	}
}

// swagger:operation GET /proposals Proposal listProposals
// ---
// summary: Returns proposals
// description: Returns list of proposals filtered by provider id
// parameters:
//   - in: query
//     name: provider_id
//     description: id of provider proposals
//     type: string
//   - in: query
//     name: service_type
//     description: the service type of the proposal. Possible values are "openvpn", "wireguard" and "noop"
//     type: string
//   - in: query
//     name: access_policy_id
//     description: the access policy id to filter the proposals by
//     type: string
//   - in: query
//     name: access_policy_source
//     description: the access policy source to filter the proposals by
//     type: string
//   - in: query
//     name: fetch_connect_counts
//     description: if set to true, fetches the connection success metrics for nodes. False by default.
//     type: boolean
// responses:
//   200:
//     description: List of proposals
//     schema:
//       "$ref": "#/definitions/ListProposalsResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	fetchConnectCounts := req.URL.Query().Get("fetch_connect_counts")

	upperTimePriceBound, err := parsePriceBound(req, "upper_time_price_bound")
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}
	lowerTimePriceBound, err := parsePriceBound(req, "lower_time_price_bound")
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	upperGBPriceBound, err := parsePriceBound(req, "upper_gb_price_bound")
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}
	lowerGBPriceBound, err := parsePriceBound(req, "lower_gb_price_bound")
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	proposals, err := pe.proposalRepository.Proposals(&proposal.Filter{
		ProviderID:          req.URL.Query().Get("provider_id"),
		ServiceType:         req.URL.Query().Get("service_type"),
		AccessPolicyID:      req.URL.Query().Get("access_policy_id"),
		AccessPolicySource:  req.URL.Query().Get("access_policy_source"),
		LowerGBPriceBound:   lowerGBPriceBound,
		UpperGBPriceBound:   upperGBPriceBound,
		LowerTimePriceBound: lowerTimePriceBound,
		UpperTimePriceBound: upperTimePriceBound,
		ExcludeUnsupported:  true,
	})

	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	proposalsRes := contract.ListProposalsResponse{Proposals: []contract.ProposalDTO{}}
	for _, p := range proposals {
		proposalsRes.Proposals = append(proposalsRes.Proposals, contract.NewProposalDTO(p))
	}

	if fetchConnectCounts == "true" {
		metrics := pe.qualityProvider.ProposalsMetrics()
		addProposalMetrics(proposalsRes.Proposals, metrics)
	}

	utils.WriteAsJSON(proposalsRes, resp)
}

func parsePriceBound(req *http.Request, key string) (*uint64, error) {
	bound := req.URL.Query().Get(key)
	if bound == "" {
		return nil, nil
	}
	upperPriceBound, err := strconv.ParseUint(req.URL.Query().Get(key), 10, 64)
	return &upperPriceBound, err
}

// AddRoutesForProposals attaches proposals endpoints to router
func AddRoutesForProposals(router *httprouter.Router, proposalRepository proposal.Repository, qualityProvider QualityFinder) {
	pe := NewProposalsEndpoint(proposalRepository, qualityProvider)
	router.GET("/proposals", pe.List)
}

// addProposalMetrics adds quality metrics to proposals.
func addProposalMetrics(proposals []contract.ProposalDTO, metrics []quality.ConnectMetric) {
	// Convert metrics slice to map for fast lookup.
	metricsMap := map[string]quality.ConnectMetric{}
	for _, m := range metrics {
		metricsMap[m.ProposalID.ProviderID+m.ProposalID.ServiceType] = m
	}

	for i, p := range proposals {
		if mc, ok := metricsMap[p.ProviderID+p.ServiceType]; ok {
			proposals[i].Metrics = &contract.ProposalMetricsDTO{ConnectCount: mc.ConnectCount}
		}
	}
}
