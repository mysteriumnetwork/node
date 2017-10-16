package dto

import (
	"testing"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/mysterium/node/money"
)

type TestServiceDefinition struct {}
func (service TestServiceDefinition) GetLocation() Location {
	return Location{}
}

type TestPaymentMethod struct {}
func (method TestPaymentMethod) GetPrice() money.Money {
	return money.Money{}
}

func TestServiceProposalSerialize(t *testing.T) {
	sp := ServiceProposal{
		Id:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "openvpn",
		ServiceDefinition: TestServiceDefinition{},
		PaymentMethodType: "PER_TIME",
		PaymentMethod:     TestPaymentMethod{},
		ProviderId:        Identity("node"),
		ProviderContacts:  []Contact{},
	}

	jsonBytes, err := json.Marshal(sp)

	expectedJson := `{
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
	assert.JSONEq(t, expectedJson, string(jsonBytes))
}
