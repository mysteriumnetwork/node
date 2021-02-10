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

import (
	"math/big"

	"github.com/mysteriumnetwork/node/session/pingpong"
)

// NewPaymentChannelDTO maps to API payment channel.
func NewPaymentChannelDTO(channel pingpong.HermesChannel) PaymentChannelDTO {
	return PaymentChannelDTO{
		ID:            channel.ChannelID,
		OwnerID:       channel.Identity.Address,
		HermesID:      channel.HermesID.Hex(),
		Earnings:      channel.UnsettledBalance(),
		EarningsTotal: channel.LifetimeBalance(),
		Beneficiary:   channel.Beneficiary.Hex(),
	}
}

// PaymentChannelDTO represents represents opened payment channel between identity and hermes.
// swagger:model PaymentChannelDTO
type PaymentChannelDTO struct {
	// Unique identifier of payment channel
	// example: 0x8fc5f7a1794dc39c6837df10613bddf1ec9810503a50306a8667f702457a739a
	ID string `json:"id"`

	// example: 0x0000000000000000000000000000000000000001
	OwnerID string `json:"owner_id"`

	// example: 0x42a537D649d6853C0a866470f2d084DA0f73b5E4
	HermesID string `json:"hermes_id"`

	// Current unsettled earnings
	// example: 19449034049997187
	Earnings *big.Int `json:"earnings"`

	// Earnings of all history
	// example: 19449034049997187
	EarningsTotal *big.Int `json:"earnings_total"`

	// Beneficiary - eth wallet address
	Beneficiary string `json:"beneficiary"`
}
