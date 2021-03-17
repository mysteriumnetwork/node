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

package mysterium

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
)

type pilvytisAPI interface {
	GetPaymentOrders(id identity.Identity) ([]pilvytis.OrderResponse, error)
}

// ListPaymentOrders list all payment orders for given identity
func (mb *MobileNode) ListPaymentOrders(id string) ([]byte, error) {
	orders, err := mb.pilvytisAPI.GetPaymentOrders(identity.FromAddress(id))
	if err != nil {
		return nil, err
	}
	return json.Marshal(orders)
}
