/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
)

// GetProposal returns the proposal for wireguard service
func GetProposal(location location.Location) market.ServiceProposal {
	marketLocation := market.Location{
		Continent: location.Continent,
		Country:   location.Country,
		City:      location.City,

		ASN:      location.ASN,
		ISP:      location.ISP,
		NodeType: location.NodeType,
	}

	return market.ServiceProposal{
		ServiceType: wg.ServiceType,
		ServiceDefinition: wg.ServiceDefinition{
			Location:          marketLocation,
			LocationOriginate: marketLocation,
		},
		PaymentMethodType: wg.PaymentMethod,
		PaymentMethod: wg.Payment{
			Price: money.NewMoney(1000000, money.CurrencyMyst),
		},
	}
}
