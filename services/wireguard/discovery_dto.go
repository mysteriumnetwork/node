/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package wireguard

import (
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
)

// ServiceType indicates "wireguard" service type
const ServiceType = "wireguard"

// ServiceDefinition structure represents "wireguard" service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location dto_discovery.Location `json:"location"`
}

// GetLocation returns geographic location of service definition provider
func (service ServiceDefinition) GetLocation() dto_discovery.Location {
	return service.Location
}

// PaymentMethod indicates payment method for Wireguard service
const PaymentMethod = "WG"

// Payment structure describes price for Wireguard service payment
type Payment struct {
	Price money.Money `json:"price"`
}

// GetPrice returns price of payment per time
func (method Payment) GetPrice() money.Money {
	return method.Price
}
