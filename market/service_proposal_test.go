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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/stretchr/testify/assert"
)

var (
	providerID      = identity.FromAddress("123456")
	providerContact = Contact{
		Type: "type1",
	}
	serviceDefinition = mockServiceDefinition{}
	paymentMethod     = mockPaymentMethod{}
)

func Test_ServiceProposal_SetProviderContact(t *testing.T) {
	proposal := ServiceProposal{ID: 123, ProviderID: "123"}
	proposal.SetProviderContact(providerID, providerContact)

	assert.Exactly(
		t,
		ServiceProposal{
			ID:               1,
			Format:           proposalFormat,
			ProviderID:       providerID.Address,
			ProviderContacts: ContactList{providerContact},
		},
		proposal,
	)
}

type mockServiceDefinition struct {
}

func (service mockServiceDefinition) GetLocation() Location {
	return Location{}
}

type mockPaymentMethod struct {
}

func (method mockPaymentMethod) GetPrice() money.Money {
	return money.Money{}
}

func (method mockPaymentMethod) GetType() string {
	return "mock"
}

func (method mockPaymentMethod) GetRate() PaymentRate {
	return PaymentRate{
		PerTime: time.Minute,
	}
}

type mockContact struct{}

func init() {
	RegisterServiceDefinitionUnserializer(
		"mock_service",
		func(rawDefinition *json.RawMessage) (ServiceDefinition, error) {
			return serviceDefinition, nil
		},
	)
	RegisterPaymentMethodUnserializer(
		"mock_payment",
		func(rawDefinition *json.RawMessage) (PaymentMethod, error) {
			return paymentMethod, nil
		},
	)
	RegisterContactUnserializer("mock_contact",
		func(rawMessage *json.RawMessage) (ContactDefinition, error) {
			return mockContact{}, nil
		},
	)
}

func Test_ServiceProposal_Serialize(t *testing.T) {
	sp := ServiceProposal{
		ID:                1,
		Format:            "format/X",
		ServiceType:       "mock_service",
		ServiceDefinition: serviceDefinition,
		PaymentMethodType: "mock_payment",
		PaymentMethod:     paymentMethod,
		ProviderID:        "node",
		ProviderContacts:  ContactList{},
	}

	jsonBytes, err := json.Marshal(sp)
	assert.Nil(t, err)

	expectedJSON := `{
	  "id": 1,
	  "format": "format/X",
	  "service_type": "mock_service",
	  "service_definition": {},
	  "payment_method_type": "mock_payment",
	  "payment_method": {},
	  "provider_id": "node",
	  "provider_contacts": []
	}`
	assert.JSONEq(t, expectedJSON, string(jsonBytes))
}

func Test_ServiceProposal_Unserialize(t *testing.T) {
	jsonData := []byte(`{
		"id": 1,
		"format": "format/X",
		"service_type": "mock_service",
		"service_definition": null,
		"payment_method_type": "mock_payment",
		"payment_method": {},
		"provider_id": "node",
		"provider_contacts": [
			{ "type" : "mock_contact" , "definition" : {}}
		]
	}`)

	var actual ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	expected := ServiceProposal{
		ID:                1,
		Format:            "format/X",
		ServiceType:       "mock_service",
		ServiceDefinition: serviceDefinition,
		PaymentMethodType: "mock_payment",
		PaymentMethod:     paymentMethod,
		ProviderID:        "node",
		ProviderContacts: ContactList{
			Contact{
				Type:       "mock_contact",
				Definition: mockContact{},
			},
		},
	}
	assert.Equal(t, expected, actual)
	assert.True(t, actual.IsSupported())
}

func Test_ServiceProposal_UnserializeUnknownService(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "unknown",
		"service_definition": {}
	}`)

	var actual ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.NoError(t, err)
	assert.Equal(t, "unknown", actual.ServiceType)
	assert.IsType(t, UnsupportedServiceDefinition{}, actual.ServiceDefinition)
}

func Test_ServiceProposal_UnserializeUnknownPaymentMethod(t *testing.T) {
	jsonData := []byte(`{
		"payment_method_type": "unknown",
		"payment_method": {}
	}`)

	var actual ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.NoError(t, err)
	assert.Equal(t, "unknown", actual.PaymentMethodType)
	assert.IsType(t, UnsupportedPaymentMethod{}, actual.PaymentMethod)
}

func Test_ServiceProposal_RegisterPaymentMethodUnserializer(t *testing.T) {
	rand := func(*json.RawMessage) (payment PaymentMethod, err error) {
		return
	}

	RegisterPaymentMethodUnserializer("testable", rand)
	_, exists := paymentMethodMap["testable"]

	assert.True(t, exists)
}

func Test_ServiceProposal_UnserializeAccessPolicy(t *testing.T) {
	jsonData := []byte(`{
		"id": 1,
		"format": "format/X",
		"service_type": "mock_service",
		"service_definition": null,
		"payment_method_type": "mock_payment",
		"payment_method": {},
		"provider_id": "node",
		"provider_contacts": [
			{ "type" : "mock_contact" , "definition" : {}}
		],
		"access_policies": [{
			"id": "verified-traffic",
			"source": "https://mysterium-oracle.mysterium.network/v1/lists/verified-traffic"
		},
		{
			"id": "dvpn-traffic",
			"source": "https://mysterium-oracle.mysterium.network/v1/lists/dvpn-traffic"
		}]
	}`)

	var actual ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	accessPolicies := []AccessPolicy{
		{
			ID:     "verified-traffic",
			Source: "https://mysterium-oracle.mysterium.network/v1/lists/verified-traffic",
		},
		{
			ID:     "dvpn-traffic",
			Source: "https://mysterium-oracle.mysterium.network/v1/lists/dvpn-traffic",
		},
	}
	expected := ServiceProposal{
		ID:                1,
		Format:            "format/X",
		ServiceType:       "mock_service",
		ServiceDefinition: serviceDefinition,
		PaymentMethodType: "mock_payment",
		PaymentMethod:     paymentMethod,
		ProviderID:        "node",
		ProviderContacts: ContactList{
			Contact{
				Type:       "mock_contact",
				Definition: mockContact{},
			},
		},
		AccessPolicies: &accessPolicies,
	}
	assert.Equal(t, expected, actual)
	assert.True(t, actual.IsSupported())
}
