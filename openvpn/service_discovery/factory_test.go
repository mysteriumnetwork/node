package service_discovery

import (
	dto "github.com/mysterium/node/openvpn/service_discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"github.com/mysterium/node/money"
)

var (
	providerId      = dto_discovery.Identity("123456")
	providerContact = dto_discovery.Contact{
		Type: "type1",
	}
	locationLTTelia = dto_discovery.Location{"LT", "Vilnius", "AS8764"}
)

func Test_NewServiceProposal(t *testing.T) {
	proposal := NewServiceProposal(providerId, providerContact)

	serviceDefinition, ok := proposal.ServiceDefinition.(dto.ServiceDefinition)
	assert.True(t, ok)
	assert.Equal(t, locationUnknown, serviceDefinition.Location)
	assert.Equal(t, locationUnknown, serviceDefinition.LocationOriginate)
}

func Test_NewServiceProposalWithLocation(t *testing.T) {
	proposal := NewServiceProposalWithLocation(providerId, providerContact, locationLTTelia)

	assert.NotNil(t, proposal)
	assert.Equal(t, 1, proposal.Id)
	assert.Equal(t, "service-proposal/v1", proposal.Format)
	assert.Equal(t, "openvpn", proposal.ServiceType)
	assert.Equal(
		t,
		dto.ServiceDefinition{
			Location:          locationLTTelia,
			LocationOriginate: locationLTTelia,
			SessionBandwidth:  83886080,
		},
		proposal.ServiceDefinition,
	)
	assert.Equal(t, dto.PAYMENT_METHOD_PER_TIME, proposal.PaymentMethodType)
	assert.Equal(
		t,
		dto.PaymentMethodPerTime{
			Price:    money.Money{12500000, money.Currency("MYST")},
			Duration: 60 * time.Minute,
		},
		proposal.PaymentMethod,
	)
	assert.Equal(t, providerId, proposal.ProviderId)
	assert.Equal(t, []dto_discovery.Contact{providerContact}, proposal.ProviderContacts)
}
