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

	"github.com/mysteriumnetwork/node/identity"
)

const (
	proposalFormat = "service-proposal/v1"
)

// ServiceProposal is top level structure which is presented to marketplace by service provider, and looked up by service consumer
// service proposal can be marked as unsupported by deserializer, because of unknown service, payment method, or contact type
type ServiceProposal struct {
	// Per provider unique serial number of service description provided
	ID int `json:"id"`

	// A version number is included in the proposal to allow extensions to the proposal format
	Format string `json:"format"`

	// Type of service type offered
	ServiceType string `json:"service_type"`

	// Qualitative service definition
	ServiceDefinition ServiceDefinition `json:"service_definition"`

	// Type of service payment method
	PaymentMethodType string `json:"payment_method_type"`

	// Service payment & usage metering definition
	PaymentMethod PaymentMethod `json:"payment_method"`

	// Unique identifier of a provider
	ProviderID string `json:"provider_id"`

	// Communication methods possible
	ProviderContacts ContactList `json:"provider_contacts"`

	// AccessPolicies represents the access controls for proposal
	AccessPolicies *[]AccessPolicy `json:"access_policies,omitempty"`
}

// AccessPolicy represents the access controls for proposal
type AccessPolicy struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

// UnmarshalJSON is custom json unmarshaler to dynamically fill in ServiceProposal values
func (proposal *ServiceProposal) UnmarshalJSON(data []byte) error {
	var jsonData struct {
		ID                int              `json:"id"`
		Format            string           `json:"format"`
		ServiceType       string           `json:"service_type"`
		ProviderID        string           `json:"provider_id"`
		PaymentMethodType string           `json:"payment_method_type"`
		ServiceDefinition *json.RawMessage `json:"service_definition"`
		PaymentMethod     *json.RawMessage `json:"payment_method"`
		ProviderContacts  *json.RawMessage `json:"provider_contacts"`
		AccessPolicies    *[]AccessPolicy  `json:"access_policies,omitempty"`
	}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	proposal.ID = jsonData.ID
	proposal.Format = jsonData.Format
	proposal.ServiceType = jsonData.ServiceType
	proposal.ProviderID = jsonData.ProviderID
	proposal.PaymentMethodType = jsonData.PaymentMethodType

	// run the service definition implementation from our registry
	proposal.ServiceDefinition = unserializeServiceDefinition(
		jsonData.ServiceType,
		jsonData.ServiceDefinition,
	)

	// run the payment method implementation from our registry
	proposal.PaymentMethod = unserializePaymentMethod(
		jsonData.PaymentMethodType,
		jsonData.PaymentMethod,
	)

	// run contact unserializer
	proposal.ProviderContacts = unserializeContacts(jsonData.ProviderContacts)

	proposal.AccessPolicies = jsonData.AccessPolicies
	return nil
}

// SetProviderContact updates service proposal description with general data
func (proposal *ServiceProposal) SetProviderContact(providerID identity.Identity, providerContact Contact) {
	proposal.Format = proposalFormat
	// TODO This will be generated later
	proposal.ID = 1
	proposal.ProviderID = providerID.Address
	proposal.ProviderContacts = ContactList{providerContact}
}

// SetAccessPolicies updates service proposal with the given AccessPolicy
func (proposal *ServiceProposal) SetAccessPolicies(ap *[]AccessPolicy) {
	proposal.AccessPolicies = ap
}

// IsSupported returns true if this service proposal can be used for connections by service consumer
// can be used as a filter to filter out all proposals which are unsupported for any reason
func (proposal *ServiceProposal) IsSupported() bool {
	if _, serviceNotSupported := proposal.ServiceDefinition.(UnsupportedServiceDefinition); serviceNotSupported {
		return false
	}
	if _, paymentNotSupported := proposal.PaymentMethod.(UnsupportedPaymentMethod); paymentNotSupported {
		return false
	}

	for _, contact := range proposal.ProviderContacts {
		if _, notSupported := contact.Definition.(UnsupportedContactType); notSupported {
			continue
		}
		//at least one is supported - we are ok
		return true
	}

	return false
}
