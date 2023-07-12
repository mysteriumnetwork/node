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
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	"github.com/mysteriumnetwork/node/services/scraping"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// QualityFinder allows to fetch proposal quality data
type QualityFinder interface {
	ProposalsQuality() []quality.ProposalQuality
}

type priceAPI interface {
	GetCurrentPrice(nodeType string, country string, serviceType string) (market.Price, error)
}

type proposalsEndpoint struct {
	proposalRepository proposalRepository
	pricer             priceAPI
	locationResolver   location.Resolver
	filterPresets      proposal.FilterPresetRepository
	natProber          natProber
}

// NewProposalsEndpoint creates and returns proposal creation endpoint
func NewProposalsEndpoint(proposalRepository proposalRepository, pricer priceAPI, locationResolver location.Resolver, filterPresetRepository proposal.FilterPresetRepository, natProber natProber) *proposalsEndpoint {
	return &proposalsEndpoint{
		proposalRepository: proposalRepository,
		pricer:             pricer,
		locationResolver:   locationResolver,
		filterPresets:      filterPresetRepository,
		natProber:          natProber,
	}
}

// swagger:operation GET /proposals Proposal listProposals
//
//	---
//	summary: Returns proposals
//	description: Returns list of proposals filtered by provider id
//	parameters:
//	  - in: query
//	    name: provider_id
//	    description: id of provider proposals
//	    type: string
//	  - in: query
//	    name: service_type
//	    description: the service type of the proposal. Possible values are "openvpn", "wireguard" and "noop"
//	    type: string
//	  - in: query
//	    name: access_policy
//	    description: the access policy id to filter the proposals by
//	    type: string
//	  - in: query
//	    name: access_policy_source
//	    description: the access policy source to filter the proposals by
//	    type: string
//	  - in: query
//	    name: country
//	    description: If given will filter proposals by node location country.
//	    type: string
//	  - in: query
//	    name: ip_type
//	    description: IP Type (residential, datacenter, etc.).
//	    type: string
//	  - in: query
//	    name: compatibility_min
//	    description: Minimum compatibility level of the proposal.
//	    type: integer
//	  - in: query
//	    name: compatibility_max
//	    description: Maximum compatibility level of the proposal.
//	    type: integer
//	  - in: query
//	    name: quality_min
//	    description: Minimum quality of the provider.
//	    type: number
//	  - in: query
//	    name: nat_compatibility
//	    description: Pick nodes compatible with NAT of specified type. Specify "auto" to probe NAT.
//	    type: string
//	responses:
//	  200:
//	    description: List of proposals
//	    schema:
//	      "$ref": "#/definitions/ListProposalsResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (pe *proposalsEndpoint) List(c *gin.Context) {
	req := c.Request
	presetID, _ := strconv.Atoi(req.URL.Query().Get("preset_id"))
	compatibilityMinQuery := req.URL.Query().Get("compatibility_min")
	compatibilityMin := 2
	if compatibilityMinQuery != "" {
		compatibilityMin, _ = strconv.Atoi(compatibilityMinQuery)
	}
	compatibilityMax, _ := strconv.Atoi(req.URL.Query().Get("compatibility_max"))
	qualityMin := func() float32 {
		f, err := strconv.ParseFloat(req.URL.Query().Get("quality_min"), 32)
		if err != nil {
			return 0
		}
		return float32(f)
	}()

	natCompatibility := nat.NATType(req.URL.Query().Get("nat_compatibility"))
	if natCompatibility == contract.AutoNATType {
		natType, err := pe.natProber.Probe(req.Context())
		if err != nil {
			natCompatibility = ""
		} else {
			natCompatibility = natType
		}
	}

	includeMonitoringFailed, _ := strconv.ParseBool(req.URL.Query().Get("include_monitoring_failed"))
	proposals, err := pe.proposalRepository.Proposals(&proposal.Filter{
		PresetID:                presetID,
		ProviderID:              req.URL.Query().Get("provider_id"),
		ServiceType:             req.URL.Query().Get("service_type"),
		AccessPolicy:            req.URL.Query().Get("access_policy"),
		AccessPolicySource:      req.URL.Query().Get("access_policy_source"),
		LocationCountry:         req.URL.Query().Get("location_country"),
		IPType:                  req.URL.Query().Get("ip_type"),
		NATCompatibility:        natCompatibility,
		CompatibilityMin:        compatibilityMin,
		CompatibilityMax:        compatibilityMax,
		QualityMin:              qualityMin,
		ExcludeUnsupported:      true,
		IncludeMonitoringFailed: includeMonitoringFailed,
	})
	if err != nil {
		c.Error(apierror.Internal("Proposal query failed: "+err.Error(), contract.ErrCodeProposalsQuery))
		return
	}

	proposalsRes := contract.ListProposalsResponse{Proposals: []contract.ProposalDTO{}}
	for _, p := range proposals {
		proposalsRes.Proposals = append(proposalsRes.Proposals, contract.NewProposalDTO(p))
	}

	utils.WriteAsJSON(proposalsRes, c.Writer)
}

// swagger:operation GET /proposals/countries Countries listCountries
//
//	---
//	summary: Returns number of proposals per country
//	description: Returns a list of countries with a number of proposals
//	parameters:
//	  - in: query
//	    name: provider_id
//	    description: id of provider proposals
//	    type: string
//	  - in: query
//	    name: service_type
//	    description: the service type of the proposal. Possible values are "openvpn", "wireguard" and "noop"
//	    type: string
//	  - in: query
//	    name: access_policy
//	    description: the access policy id to filter the proposals by
//	    type: string
//	  - in: query
//	    name: access_policy_source
//	    description: the access policy source to filter the proposals by
//	    type: string
//	  - in: query
//	    name: country
//	    description: If given will filter proposals by node location country.
//	    type: string
//	  - in: query
//	    name: ip_type
//	    description: IP Type (residential, datacenter, etc.).
//	    type: string
//	  - in: query
//	    name: compatibility_min
//	    description: Minimum compatibility level of the proposal.
//	    type: integer
//	  - in: query
//	    name: compatibility_max
//	    description: Maximum compatibility level of the proposal.
//	    type: integer
//	  - in: query
//	    name: quality_min
//	    description: Minimum quality of the provider.
//	    type: number
//	  - in: query
//	    name: nat_compatibility
//	    description: Pick nodes compatible with NAT of specified type. Specify "auto" to probe NAT.
//	    type: string
//	responses:
//	  200:
//	    description: List of countries
//	    schema:
//	      "$ref": "#/definitions/ListProposalsCountiesResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (pe *proposalsEndpoint) Countries(c *gin.Context) {
	req := c.Request

	presetID, _ := strconv.Atoi(req.URL.Query().Get("preset_id"))
	compatibilityMinQuery := req.URL.Query().Get("compatibility_min")
	compatibilityMin := 2
	if compatibilityMinQuery != "" {
		compatibilityMin, _ = strconv.Atoi(compatibilityMinQuery)
	}
	compatibilityMax, _ := strconv.Atoi(req.URL.Query().Get("compatibility_max"))
	qualityMin := func() float32 {
		f, err := strconv.ParseFloat(req.URL.Query().Get("quality_min"), 32)
		if err != nil {
			return 0
		}
		return float32(f)
	}()

	natCompatibility := nat.NATType(req.URL.Query().Get("nat_compatibility"))
	if natCompatibility == contract.AutoNATType {
		natType, err := pe.natProber.Probe(req.Context())
		if err != nil {
			natCompatibility = ""
		} else {
			natCompatibility = natType
		}
	}

	includeMonitoringFailed, _ := strconv.ParseBool(req.URL.Query().Get("include_monitoring_failed"))
	countries, err := pe.proposalRepository.Countries(&proposal.Filter{
		PresetID:                presetID,
		ProviderID:              req.URL.Query().Get("provider_id"),
		ServiceType:             req.URL.Query().Get("service_type"),
		AccessPolicy:            req.URL.Query().Get("access_policy"),
		AccessPolicySource:      req.URL.Query().Get("access_policy_source"),
		LocationCountry:         req.URL.Query().Get("location_country"),
		IPType:                  req.URL.Query().Get("ip_type"),
		NATCompatibility:        natCompatibility,
		CompatibilityMin:        compatibilityMin,
		CompatibilityMax:        compatibilityMax,
		QualityMin:              qualityMin,
		ExcludeUnsupported:      true,
		IncludeMonitoringFailed: includeMonitoringFailed,
	})
	if err != nil {
		c.Error(apierror.Internal("Proposal country query failed: "+err.Error(), contract.ErrCodeProposalsCountryQuery))
		return
	}

	utils.WriteAsJSON(countries, c.Writer)
}

// swagger:operation GET /prices/current
//
//	---
//	summary: Returns proposals
//	description: Returns list of proposals filtered by provider id
//	responses:
//	  200:
//	    description: Current proposal price
//	    schema:
//	      "$ref": "#/definitions/CurrentPriceResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (pe *proposalsEndpoint) CurrentPrice(c *gin.Context) {
	allowedServiceTypes := map[string]struct{}{
		wireguard.ServiceType:    {},
		scraping.ServiceType:     {},
		datatransfer.ServiceType: {},
		dvpn.ServiceType:         {},
	}

	serviceType := c.Request.URL.Query().Get("service_type")
	if len(serviceType) == 0 {
		serviceType = wireguard.ServiceType
	} else {
		if _, ok := allowedServiceTypes[serviceType]; !ok {
			c.Error(apierror.BadRequest("Invalid service type", contract.ErrCodeProposalsServiceType))
			return
		}
	}

	loc, err := pe.locationResolver.DetectLocation()
	if err != nil {
		c.Error(apierror.Internal("Cannot detect location", contract.ErrCodeProposalsDetectLocation))
		return
	}

	price, err := pe.pricer.GetCurrentPrice(loc.IPType, loc.Country, serviceType)
	if err != nil {
		c.Error(apierror.Internal("Cannot retrieve current prices: "+err.Error(), contract.ErrCodeProposalsPrices))
		return
	}

	utils.WriteAsJSON(contract.CurrentPriceResponse{
		ServiceType: serviceType,

		PricePerHour: price.PricePerHour,
		PricePerGiB:  price.PricePerGiB,

		PricePerHourTokens: contract.NewTokens(price.PricePerHour),
		PricePerGiBTokens:  contract.NewTokens(price.PricePerGiB),
	}, c.Writer)
}

// swagger:operation GET /v2/prices/current
//
//	---
//	summary: Returns prices
//	description: Returns prices for all service types
//	responses:
//	  200:
//	    description: Current price for service type
//	    schema:
//	      "$ref": "#/definitions/CurrentPriceResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (pe *proposalsEndpoint) CurrentPrices(c *gin.Context) {
	loc, err := pe.locationResolver.DetectLocation()
	if err != nil {
		c.Error(apierror.Internal("Cannot detect location", contract.ErrCodeProposalsDetectLocation))
		return
	}

	serviceTypes := []string{wireguard.ServiceType, scraping.ServiceType, datatransfer.ServiceType, dvpn.ServiceType}
	result := make([]contract.CurrentPriceResponse, len(serviceTypes))

	for i, serviceType := range serviceTypes {
		price, err := pe.pricer.GetCurrentPrice(loc.IPType, loc.Country, serviceType)
		if err != nil {
			c.Error(apierror.Internal("Cannot retrieve current prices: "+err.Error(), contract.ErrCodeProposalsPrices))
			return
		}

		result[i] = contract.CurrentPriceResponse{
			ServiceType:  serviceType,
			PricePerHour: price.PricePerHour,
			PricePerGiB:  price.PricePerGiB,

			PricePerHourTokens: contract.NewTokens(price.PricePerHour),
			PricePerGiBTokens:  contract.NewTokens(price.PricePerGiB),
		}
	}

	utils.WriteAsJSON(result, c.Writer)
}

// swagger:operation GET /proposals/filter-presets Proposal proposalFilterPresets
//
//	---
//	summary: Returns proposal filter presets
//	description: Returns proposal filter presets
//	responses:
//	  200:
//	    description: List of proposal filter presets
//	    schema:
//	      "$ref": "#/definitions/ListProposalFilterPresetsResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (pe *proposalsEndpoint) FilterPresets(c *gin.Context) {
	presets, err := pe.filterPresets.List()
	if err != nil {
		c.Error(apierror.Internal("Cannot list presets", contract.ErrCodeProposalsPresets))
		return
	}
	presetsRes := contract.ListProposalFilterPresetsResponse{Items: []contract.FilterPreset{}}
	for _, p := range presets.Entries {
		presetsRes.Items = append(presetsRes.Items, contract.NewFilterPreset(p))
	}
	utils.WriteAsJSON(presetsRes, c.Writer)
}

// AddRoutesForProposals attaches proposals endpoints to router
func AddRoutesForProposals(
	proposalRepository proposalRepository,
	pricer priceAPI,
	locationResolver location.Resolver,
	filterPresetRepository proposal.FilterPresetRepository,
	natProber natProber,
) func(*gin.Engine) error {
	pe := NewProposalsEndpoint(proposalRepository, pricer, locationResolver, filterPresetRepository, natProber)
	return func(e *gin.Engine) error {
		proposalGroup := e.Group("/proposals")
		{
			proposalGroup.GET("", pe.List)
			proposalGroup.GET("/filter-presets", pe.FilterPresets)
			proposalGroup.GET("/countries", pe.Countries)
		}

		e.GET("/prices/current", pe.CurrentPrice)
		e.GET("/v2/prices/current", pe.CurrentPrices)
		return nil
	}
}
