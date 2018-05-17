/*
 * Copyright (C) 2018 The Mysterium Network Authors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
	protocol string,
) dto_discovery.ServiceProposal {
	return dto_discovery.ServiceProposal{
		ID:          1,
		Format:      "service-proposal/v1",
		ServiceType: "openvpn",
		ServiceDefinition: dto.ServiceDefinition{
			Location:          serviceLocation,
			LocationOriginate: serviceLocation,
			SessionBandwidth:  dto.Bandwidth(10 * datasize.MB),
			Protocol:          protocol,
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
