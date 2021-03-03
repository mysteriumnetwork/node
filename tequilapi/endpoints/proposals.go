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
	"math/big"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// QualityFinder allows to fetch proposal quality data
type QualityFinder interface {
	ProposalsQuality() []quality.ProposalQuality
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
//     name: fetch_quality
//     description: if set to true, fetches the quality metrics for nodes. False by default.
//     type: boolean
//   - in: query
//     name: location_type
//     description: If given will filter proposals by node location type.
//     type: string
//   - in: query
//     name: location_country
//     description: If given will filter proposals by node location country.
//     type: string
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
		LocationType:        req.URL.Query().Get("location_type"),
		LocationCountry:     req.URL.Query().Get("location_country"),
		LowerGBPriceBound:   lowerGBPriceBound,
		UpperGBPriceBound:   upperGBPriceBound,
		LowerTimePriceBound: lowerTimePriceBound,
		UpperTimePriceBound: upperTimePriceBound,
		ExcludeUnsupported:  true,
		IncludeFailed:       req.URL.Query().Get("monitoring_failed") == "true",
	})
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	proposalsRes := contract.ListProposalsResponse{Proposals: []contract.ProposalDTO{}}
	for _, p := range proposals {
		proposalsRes.Proposals = append(proposalsRes.Proposals, contract.NewProposalDTO(p))
	}

	fetchQuality := req.URL.Query().Get("fetch_quality")
	if fetchQuality == "true" {
		metrics := pe.qualityProvider.ProposalsQuality()
		addProposalQuality(proposalsRes.Proposals, metrics)
	}

	utils.WriteAsJSON(proposalsRes, resp)
}

// swagger:operation GET /proposals/quality Proposal quality metrics
// ---
// summary: Returns proposals quality metrics
// description: Returns list of proposals  quality metrics
// responses:
//   200:
//     description: List of quality metrics
//     schema:
//       "$ref": "#/definitions/ProposalQualityResponse"
func (pe *proposalsEndpoint) Quality(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	quality := pe.qualityProvider.ProposalsQuality()
	utils.WriteAsJSON(contract.NewProposalQualityResponse(quality), resp)
}

func parsePriceBound(req *http.Request, key string) (*big.Int, error) {
	bound := req.URL.Query().Get(key)
	if bound == "" {
		return nil, nil
	}
	upperPriceBound, ok := new(big.Int).SetString(req.URL.Query().Get(key), 10)
	if !ok {
		return upperPriceBound, errors.New("could not parse price bound")
	}
	return upperPriceBound, nil
}

// AddRoutesForProposals attaches proposals endpoints to router
func AddRoutesForProposals(router *httprouter.Router, proposalRepository proposal.Repository, qualityProvider QualityFinder) {
	pe := NewProposalsEndpoint(proposalRepository, qualityProvider)
	router.GET("/proposals", pe.List)
	router.GET("/proposals/quality", pe.Quality)
}

// addProposalQuality adds quality metrics to proposals.
func addProposalQuality(proposals []contract.ProposalDTO, metrics []quality.ProposalQuality) {
	// Convert metrics slice to map for fast lookup.
	metricsMap := map[string]quality.ProposalQuality{}
	for _, m := range metrics {
		metricsMap[m.ProposalID.ProviderID+m.ProposalID.ServiceType] = m
	}

	for i, p := range proposals {
		if mc, ok := metricsMap[p.ProviderID+p.ServiceType]; ok {
			proposals[i].Quality = &contract.QualityMetricsDTO{
				Quality:          mc.Quality,
				MonitoringFailed: mc.MonitoringFailed,
			}
		}
	}
}
