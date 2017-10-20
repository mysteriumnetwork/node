package dto

import (
	"testing"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"encoding/json"
	"reflect"
)

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

	var actual dto.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	expected := dto.ServiceProposal{
		Id:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "openvpn",
		ServiceDefinition: ServiceDefinition{},
		PaymentMethodType: "PER_TIME",
		PaymentMethod:     PaymentMethodPerTime{},
		ProviderId:        dto.Identity("node"),
		ProviderContacts:  nil,
	}
	assert.Equal(t, expected, actual)
}

func TestServiceProposalUnserializeOpenVpnService(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": []
	}`)

	var actual dto.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.NoError(t, err)
	assert.Equal(t, "openvpn", actual.ServiceType)
	assert.Equal(t, "dto.ServiceDefinition", reflect.TypeOf(actual.ServiceDefinition).String())
}

func TestServiceProposalUnserializeUnknownService(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "unknown",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": []
	}`)

	var actual dto.ServiceProposal
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

	var actual dto.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.Nil(t, err)
	assert.Equal(t, "dto.PaymentMethodPerTime", reflect.TypeOf(actual.PaymentMethod).String())
}

func TestServiceProposalUnserializeNatsContact(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": [
			{
				"type": "nats/v1",
				"definition": {
					"topic": "test-topic"
				}
			}
		]
	}`)

	var actual dto.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.Nil(t, err)
	assert.Equal(t, "nats.ContactNATSV1", reflect.TypeOf(actual.ProviderContacts[0].Definition).String())
}

func TestServiceProposalUnserializeUnknownPaymentMethod(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "unknown",
		"payment_method": {},
		"provider_contacts": []
	}`)

	var actual dto.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.EqualError(t, err, "Payment method unserializer 'unknown' doesn't exist")
	assert.Equal(t, "unknown", actual.PaymentMethodType)
	assert.Nil(t, actual.PaymentMethod)
}

func TestServiceProposalSerialize(t *testing.T) {
	expectedJson := `{
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

	sp := dto.ServiceProposal{
		Id:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "openvpn",
		ServiceDefinition: ServiceDefinition{},
		PaymentMethodType: "PER_TIME",
		PaymentMethod:     PaymentMethodPerTime{},
		ProviderId:        dto.Identity("node"),
		ProviderContacts:  []dto.Contact{},
	}

	actualJson, err := json.Marshal(sp)
	assert.NoError(t, err)
	assert.JSONEq(t, expectedJson, string(actualJson))
}
