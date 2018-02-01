package discovery

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/money"
	"github.com/mysterium/node/openvpn/discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	providerID      = identity.FromAddress("123456")
	providerContact = dto_discovery.Contact{
		Type: "type1",
	}
	locationLTTelia = dto_discovery.Location{"LT", "Vilnius", "AS8764"}
)

func Test_NewServiceProposalWithLocation(t *testing.T) {
	proposal := NewServiceProposalWithLocation(providerID, providerContact, locationLTTelia)

	assert.NotNil(t, proposal)
	assert.Equal(t, 1, proposal.ID)
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
	assert.Equal(t, providerID.Address, proposal.ProviderID)
	assert.Equal(t, []dto_discovery.Contact{providerContact}, proposal.ProviderContacts)
}
