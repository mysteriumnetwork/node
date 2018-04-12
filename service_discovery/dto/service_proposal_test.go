package dto

import (
	"encoding/json"
	"github.com/mysterium/node/money"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestServiceDefinition struct{}

func (service TestServiceDefinition) GetLocation() Location {
	return Location{}
}

type TestPaymentMethod struct{}

func (method TestPaymentMethod) GetPrice() money.Money {
	return money.Money{}
}

func TestServiceProposalSerialize(t *testing.T) {
	sp := ServiceProposal{
		ID:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "openvpn",
		ServiceDefinition: TestServiceDefinition{},
		PaymentMethodType: "PER_TIME",
		PaymentMethod:     TestPaymentMethod{},
		ProviderID:        "node",
		ProviderContacts:  []Contact{},
	}

	jsonBytes, err := json.Marshal(sp)

	expectedJSON := `{
	  "id": 1,
	  "format": "service-proposal/v1",
	  "service_type": "openvpn",
	  "service_definition": {},
	  "payment_method_type": "PER_TIME",
	  "payment_method": {},
	  "provider_id": "node",
	  "provider_contacts": []
	}`

	assert.Nil(t, err)
	assert.JSONEq(t, expectedJSON, string(jsonBytes))
}

func TestRegisterPaymentMethodUnserializer(t *testing.T) {
	rand := func(*json.RawMessage) (payment PaymentMethod, err error) {
		return
	}

	RegisterPaymentMethodUnserializer("testable", rand)
	_, exists := paymentMethodMap["testable"]

	assert.True(t, exists)
}
