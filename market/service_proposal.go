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

package market

import (
	"encoding/json"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mysteriumnetwork/node/p2p/compat"
	"github.com/mysteriumnetwork/node/utils/validateutil"
)

const (
	proposalFormat = "service-proposal/v3"
)

// ServiceProposal is top level structure which is presented to marketplace by service provider, and looked up by service consumer
// service proposal can be marked as unsupported by deserializer, because of unknown service, payment method, or contact type
type ServiceProposal struct {
	ID int64 `json:"id"`

	// A version number is included in the proposal to allow extensions to the proposal format
	Format string `json:"format"`

	Compatibility int `json:"compatibility"`

	// Unique identifier of a provider
	ProviderID string `json:"provider_id"`

	// Type of service type offered
	ServiceType string `json:"service_type"`

	// Service location
	Location Location `json:"location"`

	// Communication methods possible
	Contacts ContactList `json:"contacts"`

	// AccessPolicies represents the access controls for proposal
	AccessPolicies *[]AccessPolicy `json:"access_policies,omitempty"`

	// Quality represents the service quality.
	Quality Quality `json:"quality"`
}

// NewProposalOpts optional params for the new proposal creation.
type NewProposalOpts struct {
	Location       *Location
	AccessPolicies []AccessPolicy
	Contacts       []Contact
	Quality        *Quality
}

// NewProposal creates a new proposal.
func NewProposal(providerID, serviceType string, opts NewProposalOpts) ServiceProposal {
	p := ServiceProposal{
		Format:         proposalFormat,
		Compatibility:  compat.Compatibility,
		ProviderID:     providerID,
		ServiceType:    serviceType,
		Location:       Location{},
		Contacts:       nil,
		AccessPolicies: nil,
	}
	if loc := opts.Location; loc != nil {
		p.Location = *loc
	}
	if ap := opts.AccessPolicies; ap != nil {
		p.AccessPolicies = &ap
	}
	if c := opts.Contacts; c != nil {
		p.Contacts = c
	}
	if q := opts.Quality; q != nil {
		p.Quality = *q
	}
	return p
}

// Validate validates the proposal.
func (proposal *ServiceProposal) Validate() error {
	return validation.ValidateStruct(proposal,
		validation.Field(&proposal.Format, validation.Required, validation.By(validateutil.StringEquals(proposalFormat))),
		validation.Field(&proposal.ProviderID, validation.Required),
		validation.Field(&proposal.ServiceType, validation.Required),
		validation.Field(&proposal.Location, validation.Required),
		validation.Field(&proposal.Contacts, validation.Required),
	)
}

// UniqueID returns unique proposal composite ID
func (proposal *ServiceProposal) UniqueID() ProposalID {
	return ProposalID{
		ProviderID:  proposal.ProviderID,
		ServiceType: proposal.ServiceType,
	}
}

// UnmarshalJSON is custom json unmarshaler to dynamically fill in ServiceProposal values
func (proposal *ServiceProposal) UnmarshalJSON(data []byte) error {
	var jsonData struct {
		ID             int64            `json:"id"`
		Format         string           `json:"format"`
		ProviderID     string           `json:"provider_id"`
		ServiceType    string           `json:"service_type"`
		Compatibility  int              `json:"compatibility"`
		Location       Location         `json:"location"`
		Contacts       *json.RawMessage `json:"contacts"`
		AccessPolicies *[]AccessPolicy  `json:"access_policies,omitempty"`
		Quality        Quality          `json:"quality"`
	}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	proposal.ID = jsonData.ID
	proposal.Format = jsonData.Format
	proposal.ProviderID = jsonData.ProviderID
	proposal.ServiceType = jsonData.ServiceType
	proposal.Compatibility = jsonData.Compatibility
	proposal.Location = jsonData.Location

	// run contact unserializer
	proposal.Contacts = unserializeContacts(jsonData.Contacts)
	proposal.AccessPolicies = jsonData.AccessPolicies
	proposal.Quality = jsonData.Quality

	return nil
}

// IsSupported returns true if this service proposal can be used for connections by service consumer
// can be used as a filter to filter out all proposals which are unsupported for any reason
func (proposal *ServiceProposal) IsSupported() bool {
	if _, ok := supportedServices[proposal.ServiceType]; !ok {
		return false
	}

	for _, contact := range proposal.Contacts {
		if _, notSupported := contact.Definition.(UnsupportedContactType); notSupported {
			continue
		}
		//at least one is supported - we are ok
		return true
	}

	return false
}

var supportedServices = make(map[string]struct{})

// RegisterServiceType registers a supported service type.
func RegisterServiceType(serviceType string) {
	supportedServices[serviceType] = struct{}{}
}
