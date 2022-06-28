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

package mysterium

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/mysteriumnetwork/node/nat"
)

// ProposalsQuery represents URL query for proposal listing
type ProposalsQuery struct {
	ProviderID      string
	ProviderIDs     []string
	ServiceType     string
	LocationCountry string

	CompatibilityMin, CompatibilityMax int
	AccessPolicy, AccessPolicySource   string

	IPType                  string
	NATCompatibility        nat.NATType
	BandwidthMin            float64
	QualityMin              float32
	IncludeMonitoringFailed bool
	PresetID                int
}

// ToURLValues converts the query to url.Values.
func (q ProposalsQuery) ToURLValues() url.Values {
	values := url.Values{}
	if q.ProviderID != "" {
		values.Set("provider_id", q.ProviderID)
	}
	for _, p := range q.ProviderIDs {
		values.Add("provider_id", p)
	}
	if q.ServiceType != "" {
		values.Set("service_type", q.ServiceType)
	}
	if q.LocationCountry != "" {
		values.Set("location_country", q.LocationCountry)
	}
	if q.IPType != "" {
		values.Set("ip_type", q.IPType)
	}
	if q.NATCompatibility != "" {
		values.Set("nat_compatibility", string(q.NATCompatibility))
	}
	if !(q.CompatibilityMin == 0 && q.CompatibilityMax == 0) {
		values.Set("compatibility_min", strconv.Itoa(q.CompatibilityMin))
		values.Set("compatibility_max", strconv.Itoa(q.CompatibilityMax))
	}
	if q.AccessPolicy != "" {
		values.Set("access_policy", q.AccessPolicy)
	}
	if q.AccessPolicySource != "" {
		values.Set("access_policy_source", q.AccessPolicySource)
	}
	if q.BandwidthMin > 0 {
		values.Set("bandwidth_min", fmt.Sprintf("%.2f", q.BandwidthMin))
	}
	if q.QualityMin != 0 {
		values.Set("quality_min", fmt.Sprintf("%.2f", q.QualityMin))
	}
	if q.IncludeMonitoringFailed {
		values.Set("include_monitoring_failed", fmt.Sprint(q.IncludeMonitoringFailed))
	}
	if q.PresetID != 0 {
		values.Set("preset_id", strconv.Itoa(q.PresetID))
	}

	return values
}
