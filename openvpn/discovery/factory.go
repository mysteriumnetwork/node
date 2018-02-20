package discovery

import (
	"github.com/mysterium/node/datasize"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/money"
	"github.com/mysterium/node/openvpn/discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"time"
)

func NewServiceProposalWithLocation(
	providerID identity.Identity,
	providerContact dto_discovery.Contact,
	serviceLocation dto_discovery.Location,
) dto_discovery.ServiceProposal {
	return dto_discovery.ServiceProposal{
		ID:          1,
		Format:      "service-proposal/v1",
		ServiceType: "openvpn",
		ServiceDefinition: dto.ServiceDefinition{
			Location:          serviceLocation,
			LocationOriginate: serviceLocation,
			SessionBandwidth:  dto.Bandwidth(10 * datasize.MB),
		},
		PaymentMethodType: dto.PaymentMethodPerTime,
		PaymentMethod: dto.PaymentPerTime{
			// 15 MYST/month = 0,5 MYST/day = 0,125 MYST/hour
			Price:    money.NewMoney(0.125, money.CURRENCY_MYST),
			Duration: 1 * time.Hour,
		},
		ProviderID:       providerID.Address,
		ProviderContacts: []dto_discovery.Contact{providerContact},
	}
}
