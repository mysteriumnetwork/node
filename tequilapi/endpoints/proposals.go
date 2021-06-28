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
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// QualityFinder allows to fetch proposal quality data
type QualityFinder interface {
	ProposalsQuality() []quality.ProposalQuality
}

type priceAPI interface {
	GetCurrentPrice(nodeType string, country string) (market.Price, error)
}

type proposalsEndpoint struct {
	proposalRepository proposalRepository
	pricer             priceAPI
	locationResolver   location.Resolver
	filterPresets      proposal.FilterPresetRepository
}

// NewProposalsEndpoint creates and returns proposal creation endpoint
func NewProposalsEndpoint(proposalRepository proposalRepository, pricer priceAPI, locationResolver location.Resolver, filterPresetRepository proposal.FilterPresetRepository) *proposalsEndpoint {
	return &proposalsEndpoint{
		proposalRepository: proposalRepository,
		pricer:             pricer,
		locationResolver:   locationResolver,
		filterPresets:      filterPresetRepository,
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
//     name: access_policy
//     description: the access policy id to filter the proposals by
//     type: string
//   - in: query
//     name: access_policy_source
//     description: the access policy source to filter the proposals by
//     type: string
//   - in: query
//     name: country
//     description: If given will filter proposals by node location country.
//     type: string
//   - in: query
//     name: ip_type
//     description: IP Type (residential, datacenter, etc.).
//     type: string
//   - in: query
//     name: compatibility_min
//     description: Minimum compatibility level of the proposal.
//     type: integer
//   - in: query
//     name: compatibility_max
//     description: Maximum compatibility level of the proposal.
//     type: integer
//   - in: query
//     name: quality_min
//     description: Minimum quality of the provider.
//     type: number
// responses:
//   200:
//     description: List of proposals
//     schema:
//       "$ref": "#/definitions/ListProposalsResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (pe *proposalsEndpoint) List(resp http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	presetID, _ := strconv.Atoi(req.URL.Query().Get("preset_id"))
	compatibilityMin, _ := strconv.Atoi(req.URL.Query().Get("compatibility_min"))
	compatibilityMax, _ := strconv.Atoi(req.URL.Query().Get("compatibility_max"))
	qualityMin := func() float32 {
		f, err := strconv.ParseFloat(req.URL.Query().Get("quality_min"), 32)
		if err != nil {
			return 0
		}
		return float32(f)
	}()

	includeMonitoringFailed, _ := strconv.ParseBool(req.URL.Query().Get("include_monitoring_failed"))
	proposals, err := pe.proposalRepository.Proposals(&proposal.Filter{
		PresetID:                presetID,
		ProviderID:              req.URL.Query().Get("provider_id"),
		ServiceType:             req.URL.Query().Get("service_type"),
		AccessPolicy:            req.URL.Query().Get("access_policy"),
		AccessPolicySource:      req.URL.Query().Get("access_policy_source"),
		LocationCountry:         req.URL.Query().Get("location_country"),
		IPType:                  req.URL.Query().Get("ip_type"),
		CompatibilityMin:        compatibilityMin,
		CompatibilityMax:        compatibilityMax,
		QualityMin:              qualityMin,
		ExcludeUnsupported:      true,
		IncludeMonitoringFailed: includeMonitoringFailed,
	})
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	proposalsRes := contract.ListProposalsResponse{Proposals: []contract.ProposalDTO{}}
	for _, p := range proposals {
		proposalsRes.Proposals = append(proposalsRes.Proposals, contract.NewProposalDTO(p))
	}

	utils.WriteAsJSON(proposalsRes, resp)
}

// swagger:operation GET /prices/current
// ---
// summary: Returns proposals
// description: Returns list of proposals filtered by provider id
// responses:
//   200:
//     description: Current proposal price
//     schema:
//       "$ref": "#/definitions/CurrentPriceResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (pe *proposalsEndpoint) CurrentPrice(resp http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	loc, err := pe.locationResolver.DetectLocation()
	if err != nil {
		utils.SendError(resp, fmt.Errorf("could not retrieve current prices: %w", err), http.StatusInternalServerError)
		return
	}

	price, err := pe.pricer.GetCurrentPrice(loc.IPType, loc.Country)
	if err != nil {
		utils.SendError(resp, fmt.Errorf("could not retrieve current prices: %w", err), http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.CurrentPriceResponse{
		PricePerHour: price.PricePerHour,
		PricePerGiB:  price.PricePerGiB,
	}, resp)
}

// swagger:operation GET /proposals/filter-presets Proposal proposalFilterPresets
// ---
// summary: Returns proposal filter presets
// description: Returns proposal filter presets
// responses:
//   200:
//     description: List of proposal filter presets
//     schema:
//       "$ref": "#/definitions/ListProposalFilterPresetsResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (pe *proposalsEndpoint) FilterPresets(resp http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	presets, err := pe.filterPresets.List()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	presetsRes := contract.ListProposalFilterPresetsResponse{Items: []contract.FilterPreset{}}
	for _, p := range presets.Entries {
		presetsRes.Items = append(presetsRes.Items, contract.NewFilterPreset(p))
	}
	utils.WriteAsJSON(presetsRes, resp)
}

// AddRoutesForProposals attaches proposals endpoints to router
func AddRoutesForProposals(router *httprouter.Router, proposalRepository proposalRepository, pricer priceAPI, locationResolver location.Resolver, filterPresetRepository proposal.FilterPresetRepository) {
	pe := NewProposalsEndpoint(proposalRepository, pricer, locationResolver, filterPresetRepository)
	router.GET("/proposals", pe.List)
	router.GET("/proposals/filter-presets", pe.FilterPresets)
	router.GET("/prices/current", pe.CurrentPrice)
}
