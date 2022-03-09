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

package contract

import (
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/terms/terms-go"
)

// TermsRequest object is accepted by terms endpoints.
// swagger:model TermsRequest
type TermsRequest struct {
	// example: false
	AgreedProvider *bool `json:"agreed_provider,omitempty"`
	// example: false
	AgreedConsumer *bool `json:"agreed_consumer,omitempty"`
	// example: 0.0.27
	AgreedVersion string `json:"agreed_version,omitempty"`
}

// TermsResponse object is returned by terms endpoints.
// swagger:model TermsResponse
type TermsResponse struct {
	// example: false
	AgreedProvider bool `json:"agreed_provider"`
	// example: false
	AgreedConsumer bool `json:"agreed_consumer"`
	// example: 0.0.27
	AgreedVersion string `json:"agreed_version"`
	// example: 0.0.27
	CurrentVersion string `json:"current_version"`
}

const (
	// TermsConsumerAgreed is the key which is used to store terms agreement
	// for consumer features.
	// This key can also be used to address the value directly in the config.
	TermsConsumerAgreed = "terms.consumer-agreed"

	// TermsProviderAgreed is the key which is used to store terms agreement
	// for provider features.
	// This key can also be used to address the value directly in the config.
	TermsProviderAgreed = "terms.provider-agreed"

	// TermsVersion is the key which is used to store terms agreement
	// version for both provider and consumer.
	// This key can also be used to address the value directly in the config.
	TermsVersion = "terms.version"
)

// NewTermsResp builds and returns terms agreement response.
func NewTermsResp() *TermsResponse {
	return &TermsResponse{
		AgreedProvider: config.Current.GetBool(TermsProviderAgreed),
		AgreedConsumer: config.Current.GetBool(TermsConsumerAgreed),
		AgreedVersion:  config.Current.GetString(TermsVersion),
		CurrentVersion: terms.TermsVersion,
	}
}

// ToMap turns a TermsRequest in to an iterable map which
// can be mapped directly to a user config.
func (t *TermsRequest) ToMap() map[string]interface{} {
	give := map[string]interface{}{}
	if t.AgreedConsumer != nil {
		give[TermsConsumerAgreed] = *t.AgreedConsumer
	}

	if t.AgreedProvider != nil {
		give[TermsProviderAgreed] = *t.AgreedProvider
	}
	give[TermsVersion] = t.AgreedVersion

	return give
}
