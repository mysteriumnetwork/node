/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package proposal

import (
	"sync"

	"github.com/mysteriumnetwork/node/core/discovery/reducer"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/nat"
)

// Filter defines all flags for proposal filtering in discovery of Mysterium Network
type Filter struct {
	PresetID                           int
	ProviderID                         string
	ProviderIDs                        []string
	ServiceType                        string
	LocationCountry                    string
	IPType                             string
	AccessPolicy, AccessPolicySource   string
	CompatibilityMin, CompatibilityMax int
	BandwidthMin                       float64
	QualityMin                         float32
	ExcludeUnsupported                 bool
	IncludeMonitoringFailed            bool
	NATCompatibility                   nat.NATType
	condition                          reducer.AndCondition
	buildOnce                          sync.Once
}

// Build ensures filter conditions are assembled
func (filter *Filter) Build() {
	filter.buildOnce.Do(func() {
		conditions := make([]reducer.AndCondition, 0)

		conditions = append(conditions, reducer.True)

		if filter.ExcludeUnsupported {
			conditions = append(conditions, reducer.Unsupported())
		}

		if filter.ProviderID != "" {
			conditions = append(conditions, reducer.Equal(reducer.ProviderID, filter.ProviderID))
		}
		if len(filter.ProviderIDs) > 0 {
			conditions = append(conditions, reducer.InString(reducer.ProviderID, filter.ProviderIDs...))
		}
		if filter.ServiceType != "" {
			conditions = append(conditions, reducer.Equal(reducer.ServiceType, filter.ServiceType))
		}
		if filter.IPType != "" {
			conditions = append(conditions, reducer.Equal(reducer.LocationType, filter.IPType))
		}
		if filter.LocationCountry != "" {
			conditions = append(conditions, reducer.Equal(reducer.LocationCountry, filter.LocationCountry))
		}
		if filter.AccessPolicy != "all" {
			if filter.AccessPolicy != "" || filter.AccessPolicySource != "" {
				conditions = append(conditions, reducer.AccessPolicy(filter.AccessPolicy, filter.AccessPolicySource))
			}
		}
		filter.condition = reducer.And(conditions...)
	})
}

// Matches return flag if filter matches given proposal
func (filter *Filter) Matches(proposal market.ServiceProposal) bool {
	filter.Build()

	return filter.condition(proposal)
}

// ToAPIQuery serialises filter to query of Mysterium API
func (filter *Filter) ToAPIQuery() mysterium.ProposalsQuery {
	query := mysterium.ProposalsQuery{
		PresetID:                filter.PresetID,
		ProviderID:              filter.ProviderID,
		ProviderIDs:             filter.ProviderIDs,
		ServiceType:             filter.ServiceType,
		LocationCountry:         filter.LocationCountry,
		IPType:                  filter.IPType,
		CompatibilityMin:        filter.CompatibilityMin,
		CompatibilityMax:        filter.CompatibilityMax,
		AccessPolicy:            filter.AccessPolicy,
		AccessPolicySource:      filter.AccessPolicySource,
		QualityMin:              filter.QualityMin,
		BandwidthMin:            filter.BandwidthMin,
		IncludeMonitoringFailed: filter.IncludeMonitoringFailed,
		NATCompatibility:        filter.NATCompatibility,
	}

	return query
}
