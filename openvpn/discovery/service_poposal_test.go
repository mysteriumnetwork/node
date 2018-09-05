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

package discovery

import (
	"encoding/json"
	"testing"

	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/openvpn"
	dto_openvpn "github.com/mysteriumnetwork/node/openvpn/discovery/dto"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

func init() {
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()
}

func TestServiceProposalUnserialize(t *testing.T) {
	jsonData := []byte(`{
		"id": 1,
		"format": "service-proposal/v1",
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_id": "node",
		"provider_contacts": []
	}`)

	var actual dto_discovery.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	expected := dto_discovery.ServiceProposal{
		ID:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "openvpn",
		ServiceDefinition: dto_openvpn.ServiceDefinition{},
		PaymentMethodType: "PER_TIME",
		PaymentMethod:     dto_openvpn.PaymentPerTime{},
		ProviderID:        "node",
		ProviderContacts:  []dto_discovery.Contact{},
	}
	assert.Equal(t, expected, actual)
}

func TestServiceProposalUnserializeUnknownService(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "unknown",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": []
	}`)

	var actual dto_discovery.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.EqualError(t, err, "Service unserializer 'unknown' doesn't exist")
	assert.Equal(t, "unknown", actual.ServiceType)
	assert.Nil(t, actual.ServiceDefinition)
}

func TestServiceProposalUnserializePerTimePaymentMethod(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": []
	}`)

	var actual dto_discovery.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.Nil(t, err)
	assert.Exactly(t, dto_openvpn.PaymentPerTime{}, actual.PaymentMethod)
}

func TestServiceProposalUnserializeUnknownPaymentMethod(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "unknown",
		"payment_method": {},
		"provider_contacts": []
	}`)

	var actual dto_discovery.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.EqualError(t, err, "Payment method unserializer 'unknown' doesn't exist")
	assert.Equal(t, "unknown", actual.PaymentMethodType)
	assert.Nil(t, actual.PaymentMethod)
}

func TestServiceProposalSerialize(t *testing.T) {
	expectedJSON := `{
		"id": 1,
		"format": "service-proposal/v1",
		"service_type": "openvpn",
		"service_definition": {
			"location": {},
			"location_originate": {}
		},
		"payment_method_type": "PER_TIME",
		"payment_method": {
			"price": {},
			"duration": 0
		},
		"provider_id": "node",
		"provider_contacts": []
	}`

	sp := dto_discovery.ServiceProposal{
		ID:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "openvpn",
		ServiceDefinition: dto_openvpn.ServiceDefinition{},
		PaymentMethodType: "PER_TIME",
		PaymentMethod:     dto_openvpn.PaymentPerTime{},
		ProviderID:        "node",
		ProviderContacts:  []dto_discovery.Contact{},
	}

	actualJSON, err := json.Marshal(sp)
	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(actualJSON))
}
