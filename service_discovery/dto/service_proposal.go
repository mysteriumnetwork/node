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

package dto

import (
	"encoding/json"
	"errors"

	"github.com/mysteriumnetwork/node/identity"
)

const (
	proposalFormat = "service-proposal/v1"
)

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
}

// SetProviderContact updates service proposal description with general data
func (proposal *ServiceProposal) SetProviderContact(providerID identity.Identity, providerContact Contact) {
	proposal.Format = proposalFormat
	// TODO This will be generated later
	proposal.ID = 1
	proposal.ProviderID = providerID.Address
	proposal.ProviderContacts = ContactList{providerContact}
}

/**
 * Service definition unserializer registry logic
 */
type ServiceDefinitionUnserializer func(*json.RawMessage) (ServiceDefinition, error)

// service definition unserializer registry
var serviceDefinitionMap map[string]ServiceDefinitionUnserializer = make(map[string]ServiceDefinitionUnserializer, 10)

func RegisterServiceDefinitionUnserializer(serviceType string, unserializer ServiceDefinitionUnserializer) {
	serviceDefinitionMap[serviceType] = unserializer
}
func unserializeServiceDefinition(serviceType string, message *json.RawMessage) (ServiceDefinition, error) {
	if method, ok := serviceDefinitionMap[serviceType]; ok {
		return method(message)
	}

	return nil, errors.New("Service unserializer '" + serviceType + "' doesn't exist")
}

/**
 * Payment method unserializer registry logic
 */
type PaymentMethodUnserializer func(*json.RawMessage) (PaymentMethod, error)

// service payment method unserializer registry
var paymentMethodMap = make(map[string]PaymentMethodUnserializer, 0)

func RegisterPaymentMethodUnserializer(paymentMethod string, unserializer func(*json.RawMessage) (PaymentMethod, error)) {
	paymentMethodMap[paymentMethod] = unserializer
}

func unserializePaymentMethod(paymentMethod string, message *json.RawMessage) (PaymentMethod, error) {
	if method, ok := paymentMethodMap[paymentMethod]; ok {
		return method(message)
	}

	return nil, errors.New("Payment method unserializer '" + paymentMethod + "' doesn't exist")
}

/**
 * Contact unserializer registry logic
 */
type ContactDefinitionUnserializer func(*json.RawMessage) (ContactDefinition, error)

// service payment method unserializer registry
var contactDefinitionMap = make(map[string]ContactDefinitionUnserializer, 0)

func RegisterContactUnserializer(paymentMethod string, unserializer func(*json.RawMessage) (ContactDefinition, error)) {
	contactDefinitionMap[paymentMethod] = unserializer
}

func unserializeContacts(message *json.RawMessage) (contactList ContactList, err error) {
	if message == nil {
		return
	}

	// get an array of raw definitions
	var contacts []struct {
		Type       string           `json:"type"`
		Definition *json.RawMessage `json:"definition"`
	}
	if err = json.Unmarshal([]byte(*message), &contacts); err != nil {
		return
	}

	contactList = make([]Contact, len(contacts))
	for index, contactItem := range contacts {
		if fn, ok := contactDefinitionMap[contactItem.Type]; ok {

			definition, er := fn(contactItem.Definition)
			if er != nil {
				return
			}

			compiled := Contact{
				Type:       contactItem.Type,
				Definition: definition,
			}

			contactList[index] = compiled
		}
	}

	return
}

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
	proposal.ServiceDefinition, _ = unserializeServiceDefinition(
		jsonData.ServiceType,
		jsonData.ServiceDefinition,
	)

	// run the payment method implementation from our registry
	proposal.PaymentMethod, _ = unserializePaymentMethod(
		jsonData.PaymentMethodType,
		jsonData.PaymentMethod,
	)

	// run contact unserializer
	proposal.ProviderContacts, _ = unserializeContacts(jsonData.ProviderContacts)

	return nil
}
