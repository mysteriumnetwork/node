package endpoints

import (
	"encoding/json"
	"github.com/mysterium/node/money"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestServiceDefinition struct{}

func (service TestServiceDefinition) GetLocation() dto_discovery.Location {
	return dto_discovery.Location{}
}

type TestPaymentMethod struct{}

func (method TestPaymentMethod) GetPrice() money.Money {
	return money.Money{}
}

func TestProposalsEndpoint_List(t *testing.T) {
	proposal := dto_discovery.ServiceProposal{
		Id:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		PaymentMethodType: "CASH",
		PaymentMethod:     TestPaymentMethod{},
		ProviderId:        "0xProviderId",
		ProviderContacts: []dto_discovery.Contact{
			dto_discovery.Contact{"phone", "what?"},
		},
	}
	proposalBytes, _ := json.Marshal(proposal)
	proposalJson := string(proposalBytes)

	discoveryApi := server.NewClientFake()
	discoveryApi.NodeRegister(proposal)

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(discoveryApi).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [`+proposalJson+`]
        }`,
		resp.Body.String(),
	)
}
