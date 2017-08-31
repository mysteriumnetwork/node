package service_discovery

import (
	dto "github.com/mysterium/node/openvpn/service_discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	nodeKey         = "123456"
	locationLTTelia = dto_discovery.Location{"LT", "Vilnius", "AS8764"}
)

func Test_NewServiceProposal(t *testing.T) {
	proposal := NewServiceProposal(nodeKey, locationLTTelia)

	assert.NotNil(t, proposal)
	assert.Equal(t, 1, proposal.Id)
	assert.Equal(t, "service-proposal/v1", proposal.Format)
	assert.Equal(t, "openvpn", proposal.ServiceType)
	assert.Equal(
		t,
		dto.ServiceDefinition{
			Location:          locationLTTelia,
			LocationOriginate: locationLTTelia,
			SessionBandwidth:  10.5,
		},
		proposal.ServiceDefinition,
	)
	assert.Equal(t, dto.PAYMENT_METHOD_PER_TIME, proposal.PaymentMethodType)
	assert.Equal(
		t,
		dto.PaymentMethodPerTime{
			Price:    dto_discovery.Price{0.125, "MYST"},
			Duration: 60 * time.Minute,
		},
		proposal.PaymentMethod,
	)
	assert.Equal(t, nodeKey, proposal.ProviderId)
	assert.Equal(
		t,
		[]dto_discovery.Contact{
			{
				Type:       dto_discovery.CONTACT_NATS_V1,
				Definition: dto_discovery.ContactNATSV1{nodeKey},
			},
		},
		proposal.ProviderContacts,
	)
}
