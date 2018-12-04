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
	"encoding/json"

	discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
)

// Bootstrap is called on program initialization time and registers various deserializers related to wireguard service
func Bootstrap() {
	discovery.RegisterServiceDefinitionUnserializer(
		ServiceType,
		func(rawDefinition *json.RawMessage) (discovery.ServiceDefinition, error) {
			var definition ServiceDefinition
			err := json.Unmarshal(*rawDefinition, &definition)

			return definition, err
		},
	)

	// TODO per time or per bytes payment methods should be defined here
	discovery.RegisterPaymentMethodUnserializer(
		PaymentMethod,
		func(rawDefinition *json.RawMessage) (discovery.PaymentMethod, error) {
			var method Payment
			err := json.Unmarshal(*rawDefinition, &method)

			return method, err
		},
	)
}
