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

	"github.com/mysteriumnetwork/node/market"
)

// CreateIdentityRequest represents JSON request for creating identity
type CreateIdentityRequest struct {
	Identity string `json:"identity"`
}

// UpdatePayoutInfoRequest represents JSON request for creating payout info request
type UpdatePayoutInfoRequest struct {
	EthAddress string `json:"payout_eth_address"`
}

// UpdateReferralInfoRequest represents JSON request for creating referral code update
type UpdateReferralInfoRequest struct {
	ReferralCode string `json:"referral_code"`
}

// UpdateEmailRequest represents JSON request for email update
type UpdateEmailRequest struct {
	Email string `json:"email"`
}

// PayoutInfoResponse represents json response for payout info
type PayoutInfoResponse struct {
	EthAddress   string `json:"eth_address"`
	ReferralCode string `json:"referral_code"`
	Email        string `json:"email"`
}

// NodeRegisterRequest represents JSON for node registration request
type NodeRegisterRequest struct {
	ServiceProposal market.ServiceProposal `json:"service_proposal"`
}

// NodeStatsRequest represents JSON request for the node session stats information
type NodeStatsRequest struct {
	NodeKey     string         `json:"node_key"`
	ServiceType string         `json:"service_type"`
	Sessions    []SessionStats `json:"sessions"`
}

// ProposalUnregisterRequest represents request JSON for unregister a single proposal
type ProposalUnregisterRequest struct {
	// Unique identifier of a provider
	ProviderID  string `json:"provider_id"`
	ServiceType string `json:"service_type"`
}

// ProposalsQuery represents URL query for proposal listing
type ProposalsQuery struct {
	ProviderID      string
	ServiceType     string
	LocationCountry string

	CompatibilityMin, CompatibilityMax int
	AccessPolicy, AccessPolicySource   string

	IPType                    string
	PriceGibMax, PriceHourMax uint64
	QualityMin                float32
}

func (q ProposalsQuery) ToURLValues() url.Values {
	values := url.Values{}
	if q.ProviderID != "" {
		values.Set("provider_id", q.ProviderID)
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
	if q.PriceGibMax != 0 {
		values.Set("price_gib_max", strconv.FormatUint(q.PriceGibMax, 10))
	}
	if q.PriceHourMax != 0 {
		values.Set("price_hour_max", strconv.FormatUint(q.PriceHourMax, 10))
	}
	if q.QualityMin != 0 {
		values.Set("quality_min", fmt.Sprintf("%.2f", q.QualityMin))
	}
	return values
}

// SessionStats mapped to json structure
type SessionStats struct {
	BytesSent       uint64 `json:"bytes_sent"`
	BytesReceived   uint64 `json:"bytes_received"`
	ProviderID      string `json:"provider_id"`
	ConsumerCountry string `json:"consumer_country"`
	ServiceType     string `json:"service_type"`
}
