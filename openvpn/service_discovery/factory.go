package service_discovery

import (
	"github.com/mysterium/node/datasize"
	dto "github.com/mysterium/node/openvpn/service_discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"time"
	"github.com/mysterium/node/money"
)

var (
	locationUnknown = dto_discovery.Location{}
)

func NewServiceProposal(
	providerId dto_discovery.Identity,
	providerContact dto_discovery.Contact,
) dto_discovery.ServiceProposal {
	return NewServiceProposalWithLocation(providerId, providerContact, locationUnknown)
}

func NewServiceProposalWithLocation(
	providerId dto_discovery.Identity,
	providerContact dto_discovery.Contact,
	nodeLocation dto_discovery.Location,
) dto_discovery.ServiceProposal {
	return dto_discovery.ServiceProposal{
		Id:          1,
		Format:      "service-proposal/v1",
		ServiceType: "openvpn",
		ServiceDefinition: dto.ServiceDefinition{
			Location:          nodeLocation,
			LocationOriginate: nodeLocation,
			SessionBandwidth:  dto.Bandwidth(10 * datasize.MB),
		},
		PaymentMethodType: dto.PAYMENT_METHOD_PER_TIME,
		PaymentMethod: dto.PaymentMethodPerTime{
			// 15 MYST/month = 0,5 MYST/day = 0,125 MYST/hour
			Price:    money.NewMoney(0.125, money.CURRENCY_MYST),
			Duration: 1 * time.Hour,
		},
		ProviderId:       dto_discovery.Identity(providerId),
		ProviderContacts: []dto_discovery.Contact{providerContact},
	}
}
