package service_discovery

import (
	dto "github.com/mysterium/node/openvpn/service_discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_NewServiceProposal(t *testing.T) {
	proposal := NewServiceProposal()

	assert.NotNil(t, proposal)
	assert.Equal(t, 1, proposal.Id)
	assert.Equal(t, "service-proposal/v1", proposal.Format)
	assert.Equal(t, "provider1", proposal.ProviderId)
	assert.Equal(t, "openvpn", proposal.ServiceType)
	assert.Equal(
		t,
		dto.ServiceDefinition{
			Location:          dto_discovery.Location{"LT", "Vilnius"},
			LocationOriginate: dto_discovery.Location{"US", "Newyork"},
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
}
