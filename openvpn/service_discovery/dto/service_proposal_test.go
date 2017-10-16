package dto

import (
	"testing"
	//"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/money"
	"github.com/mysterium/node/datasize"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestOpenVpnServiceProposalBuilderPerBytes(t *testing.T) {
	jsonData := []byte(`{
		"id": 1,
		"format": "service-proposal/v1",
		"service_type": "openvpn",
		"service_definition": {
			"location": {
				"country": "US",
				"city": "Washington DC",
				"asn" : "AS15440"
			},
			"location_originate": {
				"country": "US"
			},
			"session_bandwidth": 8
		},
		"payment_method_type": "PER_BYTES",
		"payment_method": {
			"price": {
				"amount": 50000000,
				"currency": "MYST"
			},
			"bytes": 8589934592
		},
		"provider_id": "node",
		"provider_contacts": [
			{
				"type": "test"
			}
		]
	}`)

	actual := BuildServiceProposalFromJson(jsonData)

	expected := dto.ServiceProposal{
		Id:          1,
		Format:      "service-proposal/v1",
		ServiceType: "openvpn",
		ServiceDefinition: ServiceDefinition{
			Location: dto.Location{
				Country: "US",
				City:    "Washington DC",
				ASN:     "AS15440",
			},
			LocationOriginate: dto.Location{
				Country: "US",
				City:    "",
				ASN:     "",
			},
			SessionBandwidth: 8,
		},
		PaymentMethodType: "PER_BYTES",
		PaymentMethod: PaymentMethodPerBytes{
			Price: money.Money{
				Amount:   50000000,
				Currency: money.Currency("MYST"),
			},
			Bytes: datasize.BitSize(8589934592),
		},
		ProviderId: dto.Identity("node"),
		ProviderContacts: []dto.Contact{
			{
				Type: "test",
			},
		},
	}

	assert.Equal(t, expected, actual)
}
func TestOpenVpnServiceProposalBuilderPerTime(t *testing.T) {
	jsonData := []byte(`{
		"id": 1,
		"format": "service-proposal/v1",
		"service_type": "openvpn",
		"service_definition": {
			"location": {
				"country": "US",
				"city": "Washington DC",
				"asn" : "AS15440"
			},
			"location_originate": {
				"country": "US"
			},
			"session_bandwidth": 8
		},
		"payment_method_type": "PER_TIME",
		"payment_method": {
			"price": {
				"amount": 10000000,
				"currency": "MYST"
			},
			"duration": 3600
		},
		"provider_id": "node",
		"provider_contacts": [
			{
				"type": "test"
			}
		]
	}`)

	actual := BuildServiceProposalFromJson(jsonData)

	expected := dto.ServiceProposal{
		Id:          1,
		Format:      "service-proposal/v1",
		ServiceType: "openvpn",
		ServiceDefinition: ServiceDefinition{
			Location: dto.Location{
				Country: "US",
				City:    "Washington DC",
				ASN:     "AS15440",
			},
			LocationOriginate: dto.Location{
				Country: "US",
				City:    "",
				ASN:     "",
			},
			SessionBandwidth: 8,
		},
		PaymentMethodType: "PER_TIME",
		PaymentMethod: PaymentMethodPerTime{
			Price: money.Money{
				Amount:   10000000,
				Currency: money.Currency("MYST"),
			},
			Duration: time.Duration(3600),
		},
		ProviderId: dto.Identity("node"),
		ProviderContacts: []dto.Contact{
			{
				Type: "test",
			},
		},
	}

	assert.Equal(t, expected, actual)
}
