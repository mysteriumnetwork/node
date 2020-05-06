/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package contract

// ServiceStartRequest request used to start a service.
// swagger:model ServiceStartRequestDTO
type ServiceStartRequest struct {
	// provider identity
	// required: true
	// example: 0x0000000000000000000000000000000000000002
	ProviderID string `json:"provider_id"`

	// service type. Possible values are "openvpn", "wireguard" and "noop"
	// required: true
	// example: openvpn
	Type string `json:"type"`

	// PaymentMethod describes payment options that should be used for service creation.
	// required: false
	PaymentMethod ServicePaymentMethod `json:"payment_method"`

	// access list which determines which identities will be able to receive the service
	// required: false
	AccessPolicies ServiceAccessPolicies `json:"access_policies"`

	// service options. Every service has a unique list of allowed options.
	// required: false
	// example: {"port": 1123, "protocol": "udp"}
	Options interface{} `json:"options"`
}

// ServicePaymentMethod payment parameters for service start.
// swagger:model ServicePaymentMethod
type ServicePaymentMethod struct {
	PriceGB     uint64 `json:"price_gb"`
	PriceMinute uint64 `json:"price_minute"`
}

// ServiceAccessPolicies represents the access controls for service start
// swagger:model ServiceAccessPolicies
type ServiceAccessPolicies struct {
	IDs []string `json:"ids"`
}
