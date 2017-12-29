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

var proposals = []dto_discovery.ServiceProposal{
	{
		Id:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		PaymentMethodType: "fake_payment",
		PaymentMethod:     TestPaymentMethod{},
		ProviderId:        "0xProviderId",
		ProviderContacts: []dto_discovery.Contact{
			{"phone", "what?"},
		},
	},
	{
		Id:                1,
		Format:            "service-proposal/v1",
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		PaymentMethodType: "fake_payment",
		PaymentMethod:     TestPaymentMethod{},
		ProviderId:        "other_provider",
		ProviderContacts: []dto_discovery.Contact{
			{"phone", "what?"},
		},
	},
}

func TestProposalsEndpoint_ListByNodeId(t *testing.T) {
	proposal := proposals[0]
	proposalBytes, _ := json.Marshal(proposal)

	discoveryApi := server.NewClientFake()
	for _, proposal := range proposals {
		discoveryApi.NodeRegister(proposal)
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	query := req.URL.Query()
	query.Set("nodeid", "0xProviderId")
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(discoveryApi).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [`+string(proposalBytes)+`]
        }`,
		resp.Body.String(),
	)
}

func TestProposalsEndpoint_List(t *testing.T) {
	proposalsSerializable := proposalsDto{proposals}
	proposalsBytes, _ := json.Marshal(proposalsSerializable)

	discoveryApi := server.NewClientFake()
	for _, proposal := range proposals {
		discoveryApi.NodeRegister(proposal)
	}

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
		string(proposalsBytes),
		resp.Body.String(),
	)
}
