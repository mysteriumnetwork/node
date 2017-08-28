package service_discovery

import (
	dto "github.com/mysterium/node/openvpn/service_discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"time"
)

func NewServiceProposal(nodeLocation dto_discovery.Location) dto_discovery.ServiceProposal {
	return dto_discovery.ServiceProposal{
		Id:          1,
		Format:      "service-proposal/v1",
		ProviderId:  "provider1",
		ServiceType: "openvpn",
		ServiceDefinition: dto.ServiceDefinition{
			Location:          nodeLocation,
			LocationOriginate: nodeLocation,
			SessionBandwidth:  10.5,
		},
		PaymentMethodType: dto.PAYMENT_METHOD_PER_TIME,
		PaymentMethod: dto.PaymentMethodPerTime{
			// 15 MYST/month = 0,5 MYST/day = 0,125 MYST/hour
			Price:    dto_discovery.Price{0.125, "MYST"},
			Duration: 1 * time.Hour,
		},
	}
}
